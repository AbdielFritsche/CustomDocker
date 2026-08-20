package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var (
	logsDataRoot string
	logsFollow   bool
)

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [flags] <ID_o_Nombre>",
		Short: "Muestra los registros (stdout/stderr) de un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]

			mgr := container.NewManager(logsDataRoot)
			c, err := mgr.GetContainer(idOrName)
			if err != nil {
				return fmt.Errorf("contenedor [%s] no encontrado: %w", idOrName, err)
			}

			// Ruta del archivo de log del contenedor
			logPath := filepath.Join(c.Config.BasePath, c.Config.ID, "container.log")
			file, err := os.Open(logPath)
			if err != nil {
				if os.IsNotExist(err) {
					return fmt.Errorf("no hay registros disponibles para el contenedor [%s]", idOrName)
				}
				return fmt.Errorf("error al abrir logs: %w", err)
			}
			defer file.Close()

			// 1. Imprimir logs existentes
			reader := bufio.NewReader(file)
			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					fmt.Print(line)
				}
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}
			}

			// 2. Si no se pidió follow (-f), terminar aquí
			if !logsFollow {
				return nil
			}

			// 3. Modo Follow (tail -f): seguir leyendo nuevas líneas hasta que el contenedor muera o Ctrl+C
			for {
				line, err := reader.ReadString('\n')
				if len(line) > 0 {
					fmt.Print(line)
					continue
				}

				if err == io.EOF {
					// Si el contenedor se detuvo, salimos del follow
					if c.State != container.StateRunning || (c.PID > 0 && syscall.Kill(c.PID, 0) != nil) {
						break
					}
					time.Sleep(200 * time.Millisecond)
					continue
				}

				if err != nil {
					break
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&logsDataRoot, "data-root", "/var/lib/minidocker/containers", "Ruta de almacenamiento")
	cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Seguir la salida de los logs en tiempo real")

	return cmd
}
