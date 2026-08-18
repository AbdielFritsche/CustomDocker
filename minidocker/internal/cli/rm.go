package cli

import (
	"fmt"
	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var rmDataRoot string

func newRmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rm [flags] <container_id>",
		Short: "Elimina los metadatos y almacenamiento de un contenedor detenido",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			mgr := container.NewManager(rmDataRoot)

			if err := mgr.DeleteContainer(id); err != nil {
				return err
			}

			fmt.Printf("Contenedor [%s] eliminado exitosamente de %s.\n", id, rmDataRoot)
			return nil
		},
	}

	cmd.Flags().StringVar(&rmDataRoot, "data-root", "/var/lib/minidocker/containers", "Ruta de almacenamiento de los contenedores")

	return cmd
}
