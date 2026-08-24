package cli

import (
	"context"
	"fmt"

	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <ID_o_Nombre>",
		Short: "Detiene un contenedor en ejecución",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]

			actionMsg := fmt.Sprintf("Deteniendo contenedor [%s]", idOrName)

			return decorators.WithCLIOutput(actionMsg, func() error {
				return GetAPIClient().StopContainer(context.Background(), idOrName)
			})
		},
	}
}
