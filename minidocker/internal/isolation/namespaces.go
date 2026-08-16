package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// RunParent inicia el subproceso y aplica límites de recursos
func RunParent(containerID, rootfs string, limits CgroupLimits, args []string) error {
	childArgs := append([]string{"__init__", rootfs}, args...)
	cmd := exec.Command("/proc/self/exe", childArgs...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Flags de aislamiento de Namespaces
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWIPC,
	}

	// 1. Iniciar el subproceso de forma asíncrona
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error al iniciar el subproceso: %w", err)
	}

	// 2. Aplicar límites de cgroups v2 al PID creado
	cg := NewCgroupManager(containerID)
	if err := cg.Apply(cmd.Process.Pid, limits); err != nil {
		_ = cmd.Process.Kill()
		_ = cg.Cleanup()
		return fmt.Errorf("error configurando cgroups: %w", err)
	}
	defer cg.Cleanup()

	// 3. Esperar la finalización del proceso
	return cmd.Wait()
}

// RunChild corre dentro de los nuevos namespaces aislados
func RunChild(rootfs string, command []string) error {
	// 1. Hostname aislado
	if err := syscall.Sethostname([]byte("minidocker")); err != nil {
		return fmt.Errorf("error asignando hostname: %w", err)
	}

	// 2. Cambio de sistema de archivos raíz
	if err := PivotRoot(rootfs); err != nil {
		return fmt.Errorf("error en PivotRoot: %w", err)
	}

	// 3. Montar /proc independiente
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("error montando /proc: %w", err)
	}
	defer syscall.Unmount("/proc", 0)

	// 4. Montar /dev/pts para manejo correcto de terminales
	_ = os.MkdirAll("/dev/pts", 0755)
	_ = syscall.Mount("devpts", "/dev/pts", "devpts", 0, "")
	defer syscall.Unmount("/dev/pts", 0)

	// 5. Reemplazar proceso por el comando final
	binaryPath, err := exec.LookPath(command[0])
	if err != nil {
		return fmt.Errorf("comando no encontrado dentro del rootfs: %w", err)
	}

	return syscall.Exec(binaryPath, command, os.Environ())
}