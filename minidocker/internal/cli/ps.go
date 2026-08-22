package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"minidocker/internal/api"

	"github.com/spf13/cobra"
)

func newPsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "Lista los contenedores registrados a través del daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewClient("/var/run/minidocker.sock")

			containers, err := client.GetContainers()
			if err != nil {
				return fmt.Errorf("error: %w (¿está corriendo 'minidockerd'?)", err)
			}

			if len(containers) == 0 {
				fmt.Println("No hay contenedores registrados.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
			fmt.Fprintln(w, "CONTAINER ID\tNAME\tSTATUS\tPID\tIP\tPORTS")

			for _, c := range containers {
				shortID := c.ID
				if len(shortID) > 12 {
					shortID = shortID[:12]
				}

				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
					shortID,
					c.Name,
					c.State,
					c.PID,
					c.IP,
					c.Ports,
				)
			}
			return w.Flush()
		},
	}
}
