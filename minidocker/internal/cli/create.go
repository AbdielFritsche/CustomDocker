package cli

import (
	"fmt"
	"minidocker/internal/container"
	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

var (
	createImage   string
	createMemMB   int64
	createPidsMax int64
	createPort    string
	createName    string
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [flags] <imagen> [comando]",
		Short: "Crea un nuevo contenedor sin arrancarlo",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			image := args[0]
			userCommand := args[1:]
			if len(userCommand) == 0 {
				userCommand = []string{"/bin/sh"}
			}

			actionMsg := fmt.Sprintf("Creando contenedor desde [%s]", image)

			return decorators.WithCLIOutput(actionMsg, func() error {
				opts := []container.Option{
					container.WithMemoryLimit(createMemMB * 1024 * 1024),
					container.WithPidsMax(createPidsMax),
				}
				if createName != "" {
					opts = append(opts, container.WithName(createName))
				}

				mgr := GetManager()
				c, err := mgr.CreateContainer(image, userCommand, opts...)
				if err != nil {
					return err
				}

				fmt.Printf("         -> ID asignado: %s\n", c.Config.ID)
				return nil
			})
		},
	}

	cmd.Flags().SetInterspersed(false)
	cmd.Flags().Int64VarP(&createMemMB, "memory", "m", 100, "Límite de RAM en MB")
	cmd.Flags().Int64Var(&createPidsMax, "pids-max", 20, "Límite de procesos")
	cmd.Flags().StringVarP(&createPort, "publish", "p", "", "Mapeo de puertos")
	cmd.Flags().StringVarP(&createName, "name", "n", "", "Nombre del contenedor")

	return cmd
}
