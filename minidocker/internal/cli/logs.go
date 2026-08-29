package cli

import (
	"context"
	"fmt"
	"os"

	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

var logsFollow bool

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [flags] <ID_o_Nombre>",
		Short: "Muestra los registros (stdout/stderr) de un contenedor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]
			actionMsg := fmt.Sprintf("Obteniendo registros de [%s]", idOrName)

			return decorators.WithCLIOutput(actionMsg, func() error {
				return GetAPIClient().StreamLogs(context.Background(), idOrName, logsFollow, os.Stdout)
			})
		},
	}

	cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Seguir la salida de los logs en tiempo real")

	return cmd
}
