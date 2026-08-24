package cli

import (
	"context"
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

				client := GetAPIClient()

				resp, err := client.ListContainers(context.Background())
				if err != nil {
					return err
				}

				if len(resp.Containers) == 0 {
					fmt.Println(" 			-> No hay contenedores registrados.")
					return nil
				}

				w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
				fmt.Fprintln(w, "CONTAINER ID\tIMAGE\tCOMMAND\tCREATED\tSTATUS\tPORTS\tIP\tNAMES")

				for _, c := range resp.Containers {
					cmdStr := `""`
					if len(c.Command) > 0 {
						cmdStr = fmt.Sprintf(`"%s"`, strings.Join(c.Command, " "))
						if len(cmdStr) > 20 {
							cmdStr = cmdStr[:17] + `..."`
						}
					}

					imgStr := c.Image
					if imgStr == "" {
						imgStr = "assets/alpine"
					} else {
						imgStr = filepath.Base(imgStr)
					}

					portsStr := "-"
					if c.HostPort > 0 {
						portsStr = fmt.Sprintf("0.0.0.0:%d->%d/tcp", c.HostPort, c.ContPort)
					}

					ipStr := c.IP
					if ipStr == "" {
						ipStr = "-"
					}

					createdStr := c.CreatedAt.Format("2006-01-02 15:04:05")

					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
						c.ID,
						imgStr,
						cmdStr,
						createdStr,
						c.State,
						portsStr,
						ipStr,
						c.Name,
					)
				}

				return w.Flush()
			})
		},
	}

	return cmd
}
