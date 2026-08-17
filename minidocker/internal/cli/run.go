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

			// Construir las opciones funcionales a partir de los flags
			opts := []container.Option{
				container.WithMemoryLimit(memLimitMB * 1024 * 1024),
				container.WithPidsMax(pidsMax),
			}
			if contName != "" {
				opts = append(opts, container.WithName(contName))
			}

			mgr := container.NewManager()
			c, err := mgr.CreateContainer(imagePath, userCommand, opts...)
			if err != nil {
				return fmt.Errorf("error inicializando contenedor: %w", err)
			}

			fmt.Printf("Iniciando contenedor [%s] (Mem: %dMB, PIDs: %d)...\n", c.Config.ID, memLimitMB, pidsMax)
			return mgr.RunContainer(c)
		},
	}

	// Definición de flags para el comando run
	runCmd.Flags().Int64VarP(&memLimitMB, "memory", "m", 100, "Límite de memoria RAM en MegaBytes")
	runCmd.Flags().Int64VarP(&pidsMax, "pids-max", "p", 20, "Límite máximo de procesos simultáneos")
	runCmd.Flags().StringVarP(&contName, "name", "n", "", "Nombre identificador personalizado")

	return runCmd
}