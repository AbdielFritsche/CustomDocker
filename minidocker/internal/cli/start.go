package cli

import (
	"fmt"
	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var startRootPath string

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [flags] <ID>",
		Short: "Arranca un contenedor existente conservando sus cambios",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			mgr := container.NewManager(startRootPath)

			fmt.Printf("Iniciando contenedor existente [%s]...\n", containerID)
			return mgr.StartContainer(containerID)
		},
	}

	cmd.Flags().StringVar(&startRootPath, "data-root", "/var/lib/minidocker/containers", "Ruta de almacenamiento")
	return cmd
}
