package cli

import (
	"fmt"

	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <container_id>",
		Short: "Elimina los metadatos y almacenamiento de un contenedor detenido",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			mgr := container.NewManager()

			if err := mgr.DeleteContainer(id); err != nil {
				return err
			}

			fmt.Printf("Contenedor [%s] eliminado exitosamente.\n", id)
			return nil
		},
	}
}