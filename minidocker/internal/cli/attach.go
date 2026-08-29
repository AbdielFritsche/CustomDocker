package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"minidocker/internal/api"
	"minidocker/internal/api/dto"

	"github.com/hashicorp/yamux"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var attachSocketPath string

func newAttachCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "attach [flags] <ID_o_Nombre>",
		Short: "Conecta la terminal local a los streams (stdin/stdout/stderr) de un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			return RunAttachClient(attachSocketPath, id, nil)
		},
	}

	cmd.Flags().StringVar(&attachSocketPath, "host", api.DefaultSocketPath, "Ruta al socket UNIX del daemon")
	return cmd
}

// RunAttachClient establece la conexión multiplexada con el daemon
func RunAttachClient(socketPath, containerID string, cmdOverride []string) error {
	if socketPath == "" {
		socketPath = api.DefaultSocketPath
	}

	// 1. Marcar conexión directa al socket UNIX
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("no se pudo conectar al daemon en %s: %w", socketPath, err)
	}
	defer conn.Close()

	path := fmt.Sprintf("/%s/containers/%s/attach", api.APIVersion, containerID)
	if len(cmdOverride) > 0 {
		q := url.Values{}
		for _, c := range cmdOverride {
			q.Add("cmd", c)
		}
		path += "?" + q.Encode()
	}

	req, err := http.NewRequest(http.MethodPost, path, nil)
	if err != nil {
		return fmt.Errorf("error construyendo petición: %w", err)
	}
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "yamux")

	if err := req.Write(conn); err != nil {
		return fmt.Errorf("error enviando handshake HTTP: %w", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		return fmt.Errorf("error leyendo respuesta del servidor: %w", err)
	}

	if resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("falló negociación de attach (HTTP %d): %s", resp.StatusCode, string(body))
	}

	// 3. Inicializar Yamux como cliente
	session, err := yamux.Client(conn, yamux.DefaultConfig())
	if err != nil {
		return fmt.Errorf("error iniciando sesión Yamux: %w", err)
	}
	defer session.Close()

	// 4. Aceptar los 3 streams que abre el daemon en orden: stdout, stderr, control
	stdoutStream, err := session.AcceptStream()
	if err != nil {
		return fmt.Errorf("error aceptando stdout: %w", err)
	}
	defer stdoutStream.Close()

	stderrStream, err := session.AcceptStream()
	if err != nil {
		return fmt.Errorf("error aceptando stderr: %w", err)
	}
	defer stderrStream.Close()

	controlStream, err := session.AcceptStream()
	if err != nil {
		return fmt.Errorf("error aceptando control: %w", err)
	}
	defer controlStream.Close()

	// 5. El cliente abre el stream de entrada: stdin
	stdinStream, err := session.OpenStream()
	if err != nil {
		return fmt.Errorf("error abriendo stdin stream: %w", err)
	}
	defer stdinStream.Close()

	// 6. Configurar terminal en modo Raw si es una TTY real
	fd := int(os.Stdin.Fd())
	if term.IsTerminal(fd) {
		oldState, err := term.MakeRaw(fd)
		if err == nil {
			defer term.Restore(fd, oldState)
		}

		// Enviar dimensiones iniciales
		width, height, err := term.GetSize(fd)
		if err == nil {
			_ = json.NewEncoder(controlStream).Encode(dto.ControlMessage{
				Type: "resize",
				Cols: uint16(width),
				Rows: uint16(height),
			})
		}

		// Capturar SIGWINCH para reajustar tamaño dinámicamente
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGWINCH)
		defer signal.Stop(sigChan)

		go func() {
			for range sigChan {
				w, h, err := term.GetSize(fd)
				if err == nil {
					_ = json.NewEncoder(controlStream).Encode(dto.ControlMessage{
						Type: "resize",
						Cols: uint16(w),
						Rows: uint16(h),
					})
				}
			}
		}()
	}

	// 7. Transferencia bidireccional
	errChan := make(chan error, 3)

	go func() {
		_, err := io.Copy(os.Stdout, stdoutStream)
		errChan <- err
	}()

	go func() {
		_, err := io.Copy(os.Stderr, stderrStream)
		errChan <- err
	}()

	go func() {
		_, err := io.Copy(stdinStream, os.Stdin)
		errChan <- err
	}()

	// Goroutine que espera el mensaje de terminación (exit)
	exitCodeChan := make(chan int, 1)
	go func() {
		dec := json.NewDecoder(controlStream)
		for {
			var msg dto.ControlMessage
			if err := dec.Decode(&msg); err != nil {
				return
			}
			if msg.Type == "exit" {
				exitCodeChan <- msg.Code
				return
			}
		}
	}()

	select {
	case code := <-exitCodeChan:
		if code != 0 {
			os.Exit(code)
		}
		return nil
	case <-errChan:
		return nil
	}
}
