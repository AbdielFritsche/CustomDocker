package cli

import (
	"context"
	"fmt"

	"minidocker/internal/api/dto"
	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

var (
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
				req := dto.CreateContainerRequest{
					Name:     createName,
					Image:    image,
					Command:  userCommand,
					MemoryMB: createMemMB,
					PidsMax:  createPidsMax,
					Port:     createPort,
				}

				created, err := GetAPIClient().CreateContainer(context.Background(), req)
				if err != nil {
					return err
				}

				fmt.Printf("         -> ID asignado: %s\n", created.ID)
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
