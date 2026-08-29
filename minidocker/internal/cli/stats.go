package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats [flags] <ID_o_Nombre>",
		Short: "Muestra métricas de uso de recursos (CPU, RAM, PIDs) en tiempo real",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]
			actionMsg := fmt.Sprintf("Monitoreando métricas de [%s]", idOrName)

			return decorators.WithCLIOutput(actionMsg, func() error {
				client := GetAPIClient()
				ctx := context.Background()

				initial, err := client.Stats(ctx, idOrName)
				if err != nil {
					return err
				}

				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

				prevCPU := initial.CPUUsageUsec
				prevTime := time.Now()

				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()

				fmt.Print("\033[H\033[2J")

				for {
					select {
					case <-sigChan:
						fmt.Println("\nMonitoreo finalizado.")
						return nil

					case now := <-ticker.C:
						stats, err := client.Stats(ctx, idOrName)
						if err != nil {
							fmt.Printf("\nEl contenedor [%s] ya no está en ejecución.\n", idOrName)
							return nil
						}

						deltaCPU := stats.CPUUsageUsec - prevCPU
						deltaTime := now.Sub(prevTime).Microseconds()
						cpuPercent := 0.0
						if deltaTime > 0 && deltaCPU >= 0 {
							cpuPercent = (float64(deltaCPU) / float64(deltaTime)) * 100.0
						}
						prevCPU = stats.CPUUsageUsec
						prevTime = now

						memUsedMB := float64(stats.MemUsageBytes) / (1024 * 1024)
						memLimitMB := float64(stats.MemLimitBytes) / (1024 * 1024)
						memPercent := 0.0
						memStr := fmt.Sprintf("%.2f MB / Sin Límite", memUsedMB)
						if stats.MemLimitBytes > 0 {
							memPercent = (float64(stats.MemUsageBytes) / float64(stats.MemLimitBytes)) * 100.0
							memStr = fmt.Sprintf("%.2f MB / %.2f MB", memUsedMB, memLimitMB)
						}

						pidsStr := fmt.Sprintf("%d", stats.PidsCurrent)
						if stats.PidsMax > 0 {
							pidsStr = fmt.Sprintf("%d / %d", stats.PidsCurrent, stats.PidsMax)
						}

						fmt.Print("\033[H")
						w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
						fmt.Fprintln(w, "CONTAINER ID\tNAME\tCPU %\tMEM USAGE / LIMIT\tMEM %\tPIDS")
						fmt.Fprintf(w, "%s\t%s\t%.2f%%\t%s\t%.2f%%\t%s\n",
							stats.ID, stats.Name, cpuPercent, memStr, memPercent, pidsStr)
						w.Flush()
					}
				}
			})
		},
	}
}
