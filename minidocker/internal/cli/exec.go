package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"minidocker/internal/container"

	"github.com/spf13/cobra"
)

var execDataRoot string

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec [flags] <ID_o_Nombre> <comando> [argumentos...]",
		Short: "Ejecuta un nuevo comando dentro de un contenedor en ejecución",
		Args:  cobra.MinimumNArgs(2),
		// Desactiva la intercepción de flags para que -u, -p, etc. pasen limpios al comando hijo
		DisableFlagParsing: false,
	}

	// Esto hace que Cobra deje de parsear flags en cuanto encuentra el primer argumento posicional (database)
	cmd.Flags().SetInterspersed(false)

	cmd.Flags().StringVar(&execDataRoot, "data-root", "/var/lib/minidocker/containers", "Ruta de almacenamiento de los contenedores")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		idOrName := args[0]
		userCmd := args[1:]

		mgr := container.NewManager(execDataRoot)
		c, err := mgr.GetContainer(idOrName)
		if err != nil {
			return fmt.Errorf("contenedor [%s] no encontrado: %w", idOrName, err)
		}

		if c.State != container.StateRunning || c.PID <= 0 {
			return fmt.Errorf("el contenedor [%s] no está en ejecución (estado: %s)", idOrName, c.State)
		}

		if err := syscall.Kill(c.PID, 0); err != nil {
			return fmt.Errorf("el proceso del contenedor (PID %d) ya no existe", c.PID)
		}

		cmdSerialized := strings.Join(userCmd, "\x1f")

		execCmd := exec.Command("/proc/self/exe")
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr

		execCmd.Env = append(os.Environ(),
			fmt.Sprintf("_VESSEL_EXEC_PID=%s", strconv.Itoa(c.PID)),
			fmt.Sprintf("_VESSEL_EXEC_CMD=%s", cmdSerialized),
		)

		return execCmd.Run()
	}

	return cmd
}
