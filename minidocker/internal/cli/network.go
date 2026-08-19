package cli

import (
	"fmt"
	"minidocker/internal/network"

	"github.com/spf13/cobra"
)

func newNetworkCmd() *cobra.Command {
	netCmd := &cobra.Command{
		Use:   "network",
		Short: "Administra las redes y bridges de minidocker",
	}

	rmCmd := &cobra.Command{
		Use:   "rm <network_name>",
		Short: "Elimina un bridge y red aislada (ej: br_backend o backend)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			netName := args[0]
			bridgeName := netName
			if len(netName) < 3 || netName[:3] != "br_" {
				bridgeName = "br_" + netName
			}

			if err := network.DeleteBridge(bridgeName); err != nil {
				return err
			}

			fmt.Printf("Red [%s] eliminada exitosamente del host.\n", bridgeName)
			return nil
		},
	}

	netCmd.AddCommand(rmCmd)
	return netCmd
}
