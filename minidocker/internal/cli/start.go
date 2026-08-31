package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"minidocker/internal/api/dto"
	"minidocker/pkg/decorators"
)

var (
	startCommand string
	startAttach  bool
)

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

			if startAttach {
				return decorators.WithCLIOutput(actionMsg, func() error {
					return RunAttachClient(GlobalSocketPath, containerID, cmdOverride)
				})
			}

			return decorators.WithCLIOutput(actionMsg, func() error {
				_, err := GetAPIClient().StartContainer(context.Background(), containerID, dto.StartContainerRequest{
					Command: cmdOverride,
				})
				return err
			})
		},
	}

	cmd.Flags().StringVarP(&startCommand, "command", "c", "", "Comando a ejecutar dentro del contenedor")
	cmd.Flags().BoolVarP(&startAttach, "attach", "a", false, "Arrancar y quedar adjunto a la consola (equivalente a start + attach)")

	return cmd
}
