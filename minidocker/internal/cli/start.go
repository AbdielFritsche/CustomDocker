package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"minidocker/pkg/decorators"
)

var startCommand string

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [flags] <ID_o_Nombre> [comando...]",
		Short: "Arranca un contenedor existente conservando sus cambios",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			containerID := args[0]
			var cmdOverride []string

			if startCommand != "" {
				cmdOverride = strings.Fields(startCommand)
			} else if len(args) > 1 {
				cmdOverride = args[1:]
			}

			actionMsg := fmt.Sprintf("Iniciando contenedor existente [%s]", containerID)
			if len(cmdOverride) > 0 {
				actionMsg += fmt.Sprintf(" con override de comando %v", cmdOverride)
			}

			return decorators.WithCLIOutput(actionMsg, func() error {
				return RunAttachClient(GlobalSocketPath, containerID, cmdOverride)
			})
		},
	}

	cmd.Flags().StringVarP(&startCommand, "command", "c", "", "Comando a ejecutar dentro del contenedor")

	return cmd
}
