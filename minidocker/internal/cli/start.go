package cli

import (
	"fmt"
	"strings"

	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var (
	startRootPath string
	startCommand  string
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [flags] <ID_o_Nombre> [comando...]",
		Short: "Arranca un contenedor existente conservando sus cambios",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			var cmdOverride []string

			// 1. Prioridad: flag -c (ej: -c "/bin/bash")
			if startCommand != "" {
				cmdOverride = strings.Fields(startCommand)
			} else if len(args) > 1 {
				// 2. Argumentos posicionales (ej: start db /bin/bash)
				cmdOverride = args[1:]
			}

			mgr := container.NewManager(startRootPath)
			fmt.Printf("Iniciando contenedor [%s]...\n", containerID)
			return mgr.StartContainer(containerID, cmdOverride)
		},
	}

	cmd.Flags().StringVar(&startRootPath, "data-root", "/var/lib/minidocker/containers", "Ruta de almacenamiento")
	cmd.Flags().StringVarP(&startCommand, "command", "c", "", "Comando a ejecutar dentro del contenedor")

	return cmd
}
