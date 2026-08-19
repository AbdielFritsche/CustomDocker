package cli

import (
	"fmt"
	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var stopDataRoot string

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <ID_o_Nombre>",
		Short: "Detiene uno o más contenedores en ejecución",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]
			mgr := container.NewManager(stopDataRoot)

			fmt.Printf("Deteniendo contenedor [%s]...\n", idOrName)
			if err := mgr.StopContainer(idOrName); err != nil {
				return err
			}

			fmt.Printf("Contenedor [%s] detenido exitosamente.\n", idOrName)
			return nil
		},
	}

	cmd.Flags().StringVar(&stopDataRoot, "data-root", "/var/lib/minidocker/containers", "Ruta de almacenamiento")
	return cmd
}
