package cli

import (
	"fmt"
	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var (
	createImage   string
	createMemMB   int64
	createPidsMax int64
	createPort    string
	createName    string
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [flags] <comando>",
		Short: "Crea un contenedor sin arrancarlo",
		RunE: func(cmd *cobra.Command, args []string) error {
			userCommand := args
			if len(userCommand) == 0 {
				userCommand = []string{"/bin/sh"}
			}

			opts := []container.Option{
				container.WithMemoryLimit(createMemMB * 1024 * 1024),
				container.WithPidsMax(createPidsMax),
			}
			if createName != "" {
				opts = append(opts, container.WithName(createName))
			}
			if createPort != "" {
				opts = append(opts, container.WithPortMapping(createPort))
			}

			c, err := GetManager().CreateContainer(createImage, userCommand, opts...)
			if err != nil {
				return err
			}

			fmt.Println(c.Config.ID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&createImage, "image", "i", "assets/alpine", "Ruta a la imagen base (rootfs)")
	cmd.Flags().Int64VarP(&createMemMB, "memory", "m", 100, "Límite de RAM en MB")
	cmd.Flags().Int64Var(&createPidsMax, "pids-max", 20, "Límite de procesos")
	cmd.Flags().StringVarP(&createPort, "port", "p", "", "Mapeo de puertos host:container (ej: 8080:80)")
	cmd.Flags().StringVarP(&createName, "name", "n", "", "Nombre del contenedor")

	return cmd
}
