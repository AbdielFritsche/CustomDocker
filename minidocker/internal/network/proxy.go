package network

import (
	"fmt"
	"io"
	"net"
)

// TCPProxy mantiene un socket abierto en el host y canaliza el tráfico al contenedor
type TCPProxy struct {
	listener net.Listener
	stopChan chan struct{}
}

// StartPortProxy abre un listener en todas las interfaces y canaliza tráfico bidireccional
func StartPortProxy(hostPort, containerPort int, containerIP string) (*TCPProxy, error) {
	hostAddr := fmt.Sprintf(":%d", hostPort)
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

			// Manejar cada conexión cliente en una goroutine
			go handleProxyConn(clientConn, targetAddr)
		}
	}()

	return proxy, nil
}

func handleProxyConn(client net.Conn, targetAddr string) {
	defer client.Close()

	target, err := net.Dial("tcp", targetAddr)
	if err != nil {
		return
	}
	defer target.Close()

	// Convertir a *net.TCPConn para poder cerrar escrituras independientemente
	clientTCP, ok1 := client.(*net.TCPConn)
	targetTCP, ok2 := target.(*net.TCPConn)

	done := make(chan struct{}, 2)

	// Cliente -> Contenedor
	go func() {
		_, _ = io.Copy(target, client)
		if ok2 {
			_ = targetTCP.CloseWrite()
		}
		done <- struct{}{}
	}()

	// Contenedor -> Cliente
	go func() {
		_, _ = io.Copy(client, target)
		if ok1 {
			_ = clientTCP.CloseWrite()
		}
		done <- struct{}{}
	}()

	// Esperar a que una de las dos vías termine
	<-done
}

// Close apaga el socket
func (p *TCPProxy) Close() error {
	close(p.stopChan)
	if p.listener != nil {
		return p.listener.Close()
	}
	return nil
}
