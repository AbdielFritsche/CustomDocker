package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var psDataRoot string

func newPsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps [flags]",
		Short: "Lista los contenedores registrados y su estado actual",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := container.NewManager(psDataRoot)
			baseDir := psDataRoot

			entries, err := os.ReadDir(baseDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No hay contenedores registrados en esa ruta.")
					return nil
				}
				return err
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "CONTAINER ID\tNAME\tSTATUS\tCOMMAND\tCREATED")

			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				c, err := mgr.GetContainer(entry.Name())
				if err != nil {
					continue
				}

				cmdStr := ""
				if len(c.Config.Command) > 0 {
					cmdStr = filepath.Base(c.Config.Command[0])
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					c.Config.ID,
					c.Config.Name,
					c.State,
					cmdStr,
					c.Config.CreatedAt.Format("2006-01-02 15:04:05"),
				)
			}
			return w.Flush()
		},
	}

	cmd.Flags().StringVar(&psDataRoot, "data-root", "/var/lib/minidocker/containers", "Ruta de almacenamiento")
	return cmd
}
