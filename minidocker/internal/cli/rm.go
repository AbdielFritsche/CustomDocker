package cli

import (
	"context"
	"fmt"

	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

func newRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <ID_o_Nombre>",
		Short: "Elimina los metadatos y almacenamiento de un contenedor detenido",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]

			actionMsg := fmt.Sprintf("Eliminando contenedor [%s]", idOrName)

			return decorators.WithCLIOutput(actionMsg, func() error {
				return GetAPIClient().Delete(context.Background(), idOrName)
			})
		},
	}
}
