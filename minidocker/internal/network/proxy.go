package network

import (
	"fmt"
	"io"
	"net"
	"os"
)

type TCPProxy struct {
	listener net.Listener
	stopChan chan struct{}
}

func StartPortProxy(hostPort, containerPort int, containerIP string) (*TCPProxy, error) {
	hostAddr := fmt.Sprintf("0.0.0.0:%d", hostPort)
	targetAddr := fmt.Sprintf("%s:%d", containerIP, containerPort)

	listener, err := net.Listen("tcp", hostAddr)
	if err != nil {
		return nil, fmt.Errorf("error escuchando en %s: %w", hostAddr, err)
	}

	proxy := &TCPProxy{
		listener: listener,
		stopChan: make(chan struct{}),
	}

	go func() {
		for {
			clientConn, err := listener.Accept()
			if err != nil {
				select {
				case <-proxy.stopChan:
					return
				default:
					continue
				}
			}

			go handleProxyConn(clientConn, targetAddr)
		}
	}()

	return proxy, nil
}

func handleProxyConn(client net.Conn, targetAddr string) {
	defer client.Close()

	target, err := net.Dial("tcp", targetAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Proxy Error] Fallo al conectar con %s: %v\n", targetAddr, err)
		return
	}
	defer target.Close()

	clientTCP, ok1 := client.(*net.TCPConn)
	targetTCP, ok2 := target.(*net.TCPConn)

	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(target, client)
		if ok2 {
			_ = targetTCP.CloseWrite()
		}
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(client, target)
		if ok1 {
			_ = clientTCP.CloseWrite()
		}
		done <- struct{}{}
	}()

	<-done
}

func (p *TCPProxy) Close() error {
	close(p.stopChan)
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}
