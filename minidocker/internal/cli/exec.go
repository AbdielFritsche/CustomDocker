package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [flags] <ID_o_Nombre> <comando> [argumentos...]",
		Short: "Ejecuta un nuevo comando dentro de un contenedor en ejecución",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := GetRunningContainer(args[0])
			if err != nil {
				return err
			}

			userCmd := strings.Join(args[1:], "\x1f")

			execCmd := exec.Command("/proc/self/exe")
			execCmd.Stdin = os.Stdin
			execCmd.Stdout = os.Stdout
			execCmd.Stderr = os.Stderr
			execCmd.Env = append(os.Environ(),
				fmt.Sprintf("_VESSEL_EXEC_PID=%s", strconv.Itoa(c.PID)),
				fmt.Sprintf("_VESSEL_EXEC_CMD=%s", userCmd),
			)

			return execCmd.Run()
		},
	}

	cmd.Flags().SetInterspersed(false)
	return cmd
}
