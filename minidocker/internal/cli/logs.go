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
	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

var logsFollow bool

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [flags] <ID_o_Nombre>",
		Short: "Muestra los registros (stdout/stderr) de un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]
			actionMsg := fmt.Sprintf("Obteniendo registros de [%s]", idOrName)

			return decorators.WithCLIOutput(actionMsg, func() error {
				mgr := GetManager()
				c, err := mgr.GetContainer(idOrName)
				if err != nil {
					return err
				}

				logPath := filepath.Join(c.Config.BasePath, c.Config.ID, "container.log")
				file, err := os.Open(logPath)
				if err != nil {
					if os.IsNotExist(err) {
						return fmt.Errorf("no hay registros disponibles para el contenedor [%s]", idOrName)
					}
					return fmt.Errorf("error al abrir logs: %w", err)
				}
				defer file.Close()

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

				if !logsFollow {
					return nil
				}

				// Polling estilo tail -f
				for {
					line, err := reader.ReadString('\n')
					if len(line) > 0 {
						fmt.Print(line)
						continue
					}

					if err == io.EOF {
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
			})
		},
	}

	cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Seguir la salida de los logs en tiempo real")

	return cmd
}
