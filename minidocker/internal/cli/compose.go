package cli

import (
	"fmt"
	"minidocker/internal/compose"
	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var composeFile string

func newComposeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compose [command]",
		Short: "Orquesta múltiples contenedores desde un archivo YAML",
	}

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Crea y levanta los servicios definidos en el archivo YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := container.NewManager()
			engine := compose.NewEngine(mgr)

			fmt.Printf("Procesando archivo: %s\n", composeFile)
			return engine.Up(composeFile)
		},
	}

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Detiene servicios y elimina las redes aisladas creadas",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := container.NewManager()
			engine := compose.NewEngine(mgr)
			return engine.Down(composeFile)
		},
	}
	downCmd.Flags().StringVarP(&composeFile, "file", "f", "minidocker-compose.yml", "Ruta al archivo YAML")
	upCmd.Flags().StringVarP(&composeFile, "file", "f", "minidocker-compose.yml", "Ruta al archivo YAML")

	cmd.AddCommand(downCmd)
	cmd.AddCommand(upCmd)

	return cmd
}
