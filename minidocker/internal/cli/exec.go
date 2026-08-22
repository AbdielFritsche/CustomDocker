package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"minidocker/pkg/decorators"

	"github.com/spf13/cobra"
)

var execDataRoot string

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [flags] <ID_o_Nombre> <comando> [argumentos...]",
		Short: "Ejecuta un nuevo comando dentro de un contenedor en ejecución",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			idOrName := args[0]
			userCmd := args[1:]

			actionMsg := fmt.Sprintf("Ejecutando comando en contenedor [%s]", idOrName)

			return decorators.WithCLIOutput(actionMsg, func() error {
				mgr := GetManager()
				c, err := mgr.GetContainer(idOrName)
				if err != nil {
					return err
				}

				if c.State != "running" || c.PID <= 0 || syscall.Kill(c.PID, 0) != nil {
					return fmt.Errorf("el contenedor [%s] no está en ejecución", idOrName)
				}

				childArgs := append([]string{"exec-child"}, userCmd...)
				execCmd := exec.Command("/proc/self/exe", childArgs...)
				execCmd.Stdin = os.Stdin
				execCmd.Stdout = os.Stdout
				execCmd.Stderr = os.Stderr
				execCmd.Env = append(os.Environ(), fmt.Sprintf("_VESSEL_EXEC_PID=%s", strconv.Itoa(c.PID)))

				return execCmd.Run()
			})
		},
	}

	cmd.Flags().SetInterspersed(false)
	return cmd
}
