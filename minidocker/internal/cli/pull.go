package cli

import (
	"fmt"

	"minidocker/internal/storage"
	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull <imagen:tag>",
		Short: "Descarga y ensambla una imagen oficial desde Docker Hub",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			imageName := args[0]

			actionMsg := fmt.Sprintf("Descargando imagen [%s]", imageName)

			return decorators.WithCLIOutput(actionMsg, func() error {
				path, err := storage.PullImage(imageName)
				if err != nil {
					return err
				}

				fmt.Printf("         -> Guardada en: %s\n", path)
				return nil
			})
		},
	}
}
