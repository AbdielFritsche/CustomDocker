package cli

import (
	"fmt"
	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var (
	runImage    string
	runRootPath string
	runMemMB    int64
	runPidsMax  int64
	runPort     string
	runName     string
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] <comando> [argumentos...]",
		Short: "Crea y arranca un nuevo contenedor",
		RunE: func(cmd *cobra.Command, args []string) error {
			userCommand := args
			if len(userCommand) == 0 {
				userCommand = []string{"/bin/sh"}
			}

			opts := []container.Option{
				container.WithMemoryLimit(runMemMB * 1024 * 1024),
				container.WithPidsMax(runPidsMax),
				container.WithBasePath(runRootPath),
			}
			if runName != "" {
				opts = append(opts, container.WithName(runName))
			}
			if runPort != "" {
				opts = append(opts, container.WithPortMapping(runPort))
			}

			mgr := container.NewManager(runRootPath)
			c, err := mgr.CreateContainer(runImage, userCommand, opts...)
			if err != nil {
				return fmt.Errorf("error al crear: %w", err)
			}

			fmt.Printf("Contenedor [%s] creado. Arrancando...\n", c.Config.ID)
			return mgr.StartContainer(c.Config.ID, userCommand)
		},
	}

	// Esto evita que Cobra intente parsear flags como -c, -v, -u que pertenezcan al comando interno
	cmd.Flags().SetInterspersed(false)

	cmd.Flags().StringVarP(&runImage, "image", "i", "assets/alpine", "Ruta a la imagen base")
	cmd.Flags().StringVar(&runRootPath, "data-root", "/var/lib/minidocker/containers", "Ruta de almacenamiento")
	cmd.Flags().Int64VarP(&runMemMB, "memory", "m", 100, "Límite de RAM en MB")
	cmd.Flags().Int64Var(&runPidsMax, "pids-max", 20, "Límite de procesos")
	cmd.Flags().StringVarP(&runPort, "publish", "p", "", "Mapeo de puertos host:container (ej: 8080:80)")
	cmd.Flags().StringVarP(&runName, "name", "n", "", "Nombre del contenedor")

	return cmd
}
