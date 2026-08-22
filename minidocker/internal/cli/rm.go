package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <ID_o_Nombre>",
		Short: "Elimina los metadatos y almacenamiento de un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := GetManager().DeleteContainer(id); err != nil {
				return err
			}
			fmt.Printf("Contenedor [%s] eliminado exitosamente.\n", id)
			return nil
		},
	}
}
