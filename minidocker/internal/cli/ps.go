package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

func newPsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ps [flags]",
		Short: "Lista los contenedores registrados y su estado actual",
		RunE: func(cmd *cobra.Command, args []string) error {

			actionMsg := "Listando contenedores"

			return decorators.WithCLIOutput(actionMsg, func() error {

				mgr := GetManager()

				entries, err := os.ReadDir(GlobalDataRoot)
				if err != nil {
					if os.IsNotExist(err) {
						fmt.Println("         -> No hay contenedores registrados en esa ruta.")
						return nil
					}
					return err
				}

				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintln(w, "CONTAINER ID\tIMAGE\tCOMMAND\tCREATED\tSTATUS\tPORTS\tIP\tNAMES")

				for _, entry := range entries {
					if !entry.IsDir() {
						continue
					}

					c, err := mgr.GetContainer(entry.Name())
					if err != nil {
						continue
					}

					cmdStr := `""`
					if len(c.Config.Command) > 0 {
						cmdStr = fmt.Sprintf(`"%s"`, strings.Join(c.Config.Command, " "))
						if len(cmdStr) > 20 {
							cmdStr = cmdStr[:17] + `..."`
						}
					}

					imgStr := c.Config.Image
					if imgStr == "" {
						imgStr = "assets/alpine"
					} else {
						imgStr = filepath.Base(imgStr)
					}

					portsStr := "-"
					if c.Config.PortMapping != nil && c.Config.PortMapping.HostPort > 0 {
						portsStr = fmt.Sprintf("0.0.0.0:%d->%d/tcp", c.Config.PortMapping.HostPort, c.Config.PortMapping.ContainerPort)
					}

					ipStr := c.Config.IP
					if ipStr == "" {
						ipStr = "-"
					}

					createdStr := c.Config.CreatedAt.Format("2006-01-02 15:04:05")

					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
						c.Config.ID[:12],
						imgStr,
						cmdStr,
						createdStr,
						c.State,
						portsStr,
						ipStr,
						c.Config.Name,
					)
				}

				return w.Flush()
			})
		},
	}

	return cmd
}
