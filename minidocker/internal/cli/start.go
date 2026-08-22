package cli

import (
	"fmt"
	"time"

	"minidocker/internal/api"
	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

var startOverrideCmd string

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [flags] <ID_o_Nombre>",
		Short: "Inicia un contenedor existente a través del daemon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]
			client := api.NewClient("/var/run/minidocker.sock")

			var override []string
			if startOverrideCmd != "" {
				override = []string{startOverrideCmd}
			}

			// 1. Decorador visual: Enviar orden al Daemon
			err := decorators.WithLogging(fmt.Sprintf("Iniciando contenedor [%s]", idOrName), func() error {
				return client.StartContainer(idOrName, override...)
			})
			if err != nil {
				return err
			}

			// 2. Decorador visual: Confirmar estado activo de namespaces y proxy
			return decorators.WithLogging(fmt.Sprintf("Verificando estado activo para [%s]", idOrName), func() error {
				for i := 0; i < 15; i++ {
					time.Sleep(300 * time.Millisecond)
					containers, err := client.GetContainers()
					if err != nil {
						continue
					}
					for _, c := range containers {
						if c.ID == idOrName || c.Name == idOrName {
							if c.State == "running" {
								return nil
							}
							if c.State == "failed" {
								return fmt.Errorf("el contenedor terminó con estado failed")
							}
						}
					}
				}
				return fmt.Errorf("timeout esperando estado running")
			})
		},
	}

	cmd.Flags().StringVarP(&startOverrideCmd, "command", "c", "", "Comando opcional de sobreescritura")
	return cmd
}
