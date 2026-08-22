package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "minidocker",
	Short: "MiniDocker - Runtime de contenedores de bajo nivel en Linux",
	Long: `MiniDocker es un motor de contenedores modular construido desde cero en Go.
Implementa aislamiento con Namespaces, gobernanza con cgroups v2 y almacenamiento OverlayFS.`,
}

// Execute inicia el árbol de ejecución de comandos
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	// PersistentFlags queda disponible para run, create, ps, stop, logs, etc.
	rootCmd.PersistentFlags().StringVar(&globalDataRoot, "data-root", "/var/lib/minidocker/containers", "Ruta de almacenamiento de los contenedores")

	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newCreateCmd())
	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newPsCmd())
	rootCmd.AddCommand(newRmCmd())
	rootCmd.AddCommand(newPullCmd())
	rootCmd.AddCommand(newComposeCmd())
	rootCmd.AddCommand(newStopCmd())
	rootCmd.AddCommand(newNetworkCmd())
	rootCmd.AddCommand(newExecCmd())
	rootCmd.AddCommand(newLogsCmd())
	rootCmd.AddCommand(newStatsCmd())
}
