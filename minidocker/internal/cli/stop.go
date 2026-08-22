package cli

import (
	"fmt"
	"minidocker/internal/api"

	"github.com/spf13/cobra"
)

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <ID_o_Nombre>",
		Short: "Detiene un contenedor en ejecución a través del daemon",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]
			client := api.NewClient("/var/run/minidocker.sock")

			fmt.Printf("Enviando orden de detención para [%s]...\n", idOrName)
			if err := client.StopContainer(idOrName); err != nil {
				return err
			}

			fmt.Printf("Contenedor [%s] detenido exitosamente.\n", idOrName)
			return nil
		},
	}
}
