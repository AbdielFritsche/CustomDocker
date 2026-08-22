package cli

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"minidocker/internal/container"
	"minidocker/internal/isolation"
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
				mgr := GetManager()
				c, err := mgr.GetContainer(idOrName)
				if err != nil {
					return err
				}

				if c.State != container.StateRunning || c.PID <= 0 {
					return fmt.Errorf("el contenedor [%s] no está en ejecución", idOrName)
				}

				cg := isolation.NewCgroupManager(c.Config.ID)

				sigChan := make(chan os.Signal, 1)
				signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

				prevCPU := int64(0)
				prevTime := time.Now()

				if initialStats, err := cg.ReadStats(); err == nil {
					prevCPU = initialStats.CPUUsageUsec
				}

				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()

				fmt.Print("\033[H\033[2J")

				for {
					select {
					case <-sigChan:
						fmt.Println("\nMonitoreo finalizado.")
						return nil

					case now := <-ticker.C:
						if err := syscall.Kill(c.PID, 0); err != nil {
							fmt.Printf("\nEl contenedor [%s] ha finalizado.\n", c.Config.Name)
							return nil
						}

						stats, err := cg.ReadStats()
						if err != nil {
							return fmt.Errorf("error leyendo métricas: %w", err)
						}

						deltaCPU := stats.CPUUsageUsec - prevCPU
						deltaTime := now.Sub(prevTime).Microseconds()
						cpuPercent := 0.0
						if deltaTime > 0 && deltaCPU >= 0 {
							cpuPercent = (float64(deltaCPU) / float64(deltaTime)) * 100.0
						}
						prevCPU = stats.CPUUsageUsec
						prevTime = now

						memUsedMB := float64(stats.MemoryUsageBytes) / (1024 * 1024)
						memLimitMB := float64(stats.MemoryLimitBytes) / (1024 * 1024)
						memPercent := 0.0
						memStr := fmt.Sprintf("%.2f MB / Sin Límite", memUsedMB)
						if stats.MemoryLimitBytes > 0 {
							memPercent = (float64(stats.MemoryUsageBytes) / float64(stats.MemoryLimitBytes)) * 100.0
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
							c.Config.ID,
							c.Config.Name,
							cpuPercent,
							memStr,
							memPercent,
							pidsStr,
						)
						w.Flush()
					}
				}
			})
		},
	}
}
