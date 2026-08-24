package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"minidocker/internal/api"
	"minidocker/internal/api/dto"
	"minidocker/pkg/decorators"

	"github.com/hashicorp/yamux"
	"github.com/spf13/cobra"
)

var execSocketPath string

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [flags] <ID_o_Nombre> <comando> [argumentos...]",
		Short: "Ejecuta un nuevo comando dentro de un contenedor en ejecución",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]
			userCmd := args[1:]

			actionMsg := fmt.Sprintf("Ejecutando comando en contenedor [%s]", idOrName)
			return decorators.WithCLIOutput(actionMsg, func() error {
				return RunExecClient(execSocketPath, idOrName, userCmd)
			})
		},
	}

	cmd.Flags().SetInterspersed(false)
	cmd.Flags().StringVar(&execSocketPath, "socket", api.DefaultSocketPath, "Ruta del socket UNIX de minidockerd")
	return cmd
}

// RunExecClient establece la conexión multiplexada con el daemon para exec
func RunExecClient(socketPath, containerID string, userCmd []string) error {
	if socketPath == "" {
		socketPath = api.DefaultSocketPath
	}

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("no se pudo conectar al daemon en %s: %w", socketPath, err)
	}
	defer conn.Close()

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("/%s/containers/%s/exec", api.APIVersion, containerID), nil)
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
		return fmt.Errorf("falló negociación de exec (HTTP %d): %s", resp.StatusCode, string(body))
	}

	session, err := yamux.Client(conn, yamux.DefaultConfig())
	if err != nil {
		return fmt.Errorf("error iniciando sesión Yamux: %w", err)
	}
	defer session.Close()

	// Mismo orden que el server: aceptar stdout, stderr, control; abrir stdin.
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

	stdinStream, err := session.OpenStream()
	if err != nil {
		return fmt.Errorf("error abriendo stdin: %w", err)
	}
	defer stdinStream.Close()

	// Enviar el comando como primer mensaje del controlStream
	if err := json.NewEncoder(controlStream).Encode(dto.ExecRequest{Command: userCmd}); err != nil {
		return fmt.Errorf("error enviando comando: %w", err)
	}

	errChan := make(chan error, 3)
	go func() { _, e := io.Copy(os.Stdout, stdoutStream); errChan <- e }()
	go func() { _, e := io.Copy(os.Stderr, stderrStream); errChan <- e }()
	go func() { _, e := io.Copy(stdinStream, os.Stdin); errChan <- e }()

	exitCodeChan := make(chan int, 1)
	go func() {
		dec := json.NewDecoder(controlStream)
		for {
			var msg dto.ControlMessage
			if err := dec.Decode(&msg); err != nil {
				return
			}
			if msg.Type == "exit" || msg.Type == "error" {
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
