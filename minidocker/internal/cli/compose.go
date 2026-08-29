package cli

import (
	"context"
	"fmt"
	"os"

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
			data, err := os.ReadFile(composeFile)
			if err != nil {
				return fmt.Errorf("error leyendo archivo compose [%s]: %w", composeFile, err)
			}
			fmt.Printf("Procesando archivo: %s\n", composeFile)
			return GetAPIClient().ComposeUp(context.Background(), string(data))
		},
	}

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Detiene servicios y elimina las redes aisladas creadas",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := os.ReadFile(composeFile)
			if err != nil {
				return fmt.Errorf("error leyendo archivo compose [%s]: %w", composeFile, err)
			}
			return GetAPIClient().ComposeDown(context.Background(), string(data))
		},
	}
	downCmd.Flags().StringVarP(&composeFile, "file", "f", "minidocker-compose.yml", "Ruta al archivo YAML")
	upCmd.Flags().StringVarP(&composeFile, "file", "f", "minidocker-compose.yml", "Ruta al archivo YAML")

	cmd.AddCommand(downCmd)
	cmd.AddCommand(upCmd)
	return cmd
}
