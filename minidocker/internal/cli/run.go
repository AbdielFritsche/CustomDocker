package cli

import (
	"fmt"
	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var (
	memLimitMB int64
	pidsMax    int64
	contName   string
	portMap    string
)

func newRunCmd() *cobra.Command {
	runCmd := &cobra.Command{
		Use:   "run [flags] <comando>",
		Short: "Ejecuta un comando dentro de un nuevo contenedor aislado",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			imagePath := "assets/alpine"
			userCommand := args
			if len(userCommand) == 0 {
				userCommand = []string{"/bin/sh"}
			}

			opts := []container.Option{
				container.WithMemoryLimit(memLimitMB * 1024 * 1024),
				container.WithPidsMax(pidsMax),
			}
			if contName != "" {
				opts = append(opts, container.WithName(contName))
			}
			if portMap != "" {
				opts = append(opts, container.WithPortMapping(portMap))
			}

			mgr := container.NewManager()
			c, err := mgr.CreateContainer(imagePath, userCommand, opts...)
			if err != nil {
				return fmt.Errorf("error inicializando contenedor: %w", err)
			}

			fmt.Printf("Iniciando contenedor [%s] (Mem: %dMB, PIDs: %d, Port: %s)...\n", c.Config.ID, memLimitMB, pidsMax, portMap)
			return mgr.RunContainer(c)
		},
	}

	runCmd.Flags().Int64VarP(&memLimitMB, "memory", "m", 100, "Límite de memoria RAM en MegaBytes")
	runCmd.Flags().Int64Var(&pidsMax, "pids-max", 20, "Límite máximo de procesos simultáneos")
	runCmd.Flags().StringVarP(&contName, "name", "n", "", "Nombre identificador personalizado")
	runCmd.Flags().StringVarP(&portMap, "publish", "p", "", "Mapeo de puertos host:container (ej. 8080:80)")

	return runCmd
}
