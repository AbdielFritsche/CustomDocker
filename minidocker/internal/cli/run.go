package cli

import (
	"context"
	"fmt"

	"minidocker/internal/api"
	"minidocker/internal/api/dto"
	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

var (
	runImage   string
	runMemMB   int64
	runPidsMax int64
	runPort    string
	runDetach  bool
	runName    string
	runHost    string
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] <imagen> [comando] [argumentos...]",
		Short: "Crea y arranca un nuevo contenedor",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			image := args[0]
			userCommand := args[1:]

			if len(userCommand) == 0 {
				userCommand = []string{"/bin/sh"}
			}

			actionMsg := fmt.Sprintf("Creando y arrancando contenedor desde [%s]", image)

			client := api.NewClient(runHost)
			ctx := context.Background()

			return decorators.WithCLIOutput(actionMsg, func() error {
				req := dto.CreateContainerRequest{
					Name:     runName,
					Image:    image,
					Command:  userCommand,
					MemoryMB: runMemMB,
					PidsMax:  runPidsMax,
					Port:     runPort,
				}

				created, err := client.CreateContainer(ctx, req)
				if err != nil {
					return fmt.Errorf("error creando contenedor: %w", err)
				}

				// 1. Modo background (-d): Iniciar vía REST asíncrono
				if runDetach {
					fmt.Println(created.ID)
					_, err := client.StartContainer(ctx, created.ID, dto.StartContainerRequest{
						Command: userCommand,
						Attach:  false,
					})
					return err
				}

				// 2. Modo interactivo: Multiplexar sesión Yamux sobre el socket UNIX
				return RunAttachClient(runHost, created.ID)
			})
		},
	}

	cmd.Flags().SetInterspersed(false)
	cmd.Flags().Int64VarP(&runMemMB, "memory", "m", 100, "Límite de RAM en MB")
	cmd.Flags().Int64Var(&runPidsMax, "pids-max", 20, "Límite de procesos")
	cmd.Flags().StringVarP(&runPort, "publish", "p", "", "Mapeo de puertos host:container (ej: 8080:80)")
	cmd.Flags().StringVarP(&runName, "name", "n", "", "Nombre del contenedor")
	cmd.Flags().BoolVarP(&runDetach, "detach", "d", false, "Ejecutar contenedor en segundo plano")
	cmd.Flags().StringVar(&runHost, "host", api.DefaultSocketPath, "Ruta al socket UNIX de minidockerd")

	return cmd
}
