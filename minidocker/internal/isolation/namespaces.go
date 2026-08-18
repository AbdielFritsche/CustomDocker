package isolation

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"io"
	"minidocker/internal/network"
)

// RunParent inicia el subproceso y aplica límites de recursos
func RunParent(containerID, rootfs string, limits CgroupLimits, args []string) error {
	// 1. Inicializar el Linux Bridge en el host
	if err := network.SetupBridge(); err != nil {
		return fmt.Errorf("error configurando bridge: %w", err)
	}

	// 2. Crear Pipe de sincronización (padre -> hijo)
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("error creando pipe de sincronización: %w", err)
	}
	defer pipeReader.Close()

	childArgs := append([]string{"__init__", rootfs}, args...)
	cmd := exec.Command("/proc/self/exe", childArgs...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Pasar el extremo de lectura del pipe al subproceso hijo (Descriptor extra #3)
	cmd.ExtraFiles = []*os.File{pipeReader}

	// Flags de aislamiento: UTS, PID, Mount, IPC y NET (Red independiente)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET,
	}

	// 3. Iniciar el subproceso de forma asíncrona
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error al iniciar el subproceso: %w", err)
	}

	pid := cmd.Process.Pid

	// 4. Aplicar límites de Cgroups v2
	cg := NewCgroupManager(containerID)
	if err := cg.Apply(pid, limits); err != nil {
		_ = cmd.Process.Kill()
		_ = cg.Cleanup()
		return fmt.Errorf("error configurando cgroups: %w", err)
	}
	defer cg.Cleanup()

	// 5. Configurar interfaces virtuales de red (veth pair + IP privada)
	containerIP := "172.19.0.2/16"
	if err := network.SetupContainerNetwork(pid, containerIP); err != nil {
		_ = cmd.Process.Kill()
		network.CleanupNetwork(pid)
		return fmt.Errorf("error configurando red del contenedor: %w", err)
	}
	defer network.CleanupNetwork(pid)

	// 6. Desbloquear al hijo escribiendo en el pipe
	_, _ = pipeWriter.Write([]byte("ready"))
	_ = pipeWriter.Close()

	// 7. Esperar a que el contenedor termine
	return cmd.Wait()
}

// RunChild corre dentro de los nuevos namespaces aislados
func RunChild(rootfs string, command []string) error {
	// 1. Esperar la señal de sincronización de red desde el padre
	// ExtraFiles[0] corresponde al file descriptor 3
	syncPipe := os.NewFile(uintptr(3), "syncPipe")
	buf := make([]byte, 5)
	_, _ = io.ReadFull(syncPipe, buf)
	_ = syncPipe.Close()

	// 2. Establecer Hostname aislado
	if err := syscall.Sethostname([]byte("minidocker")); err != nil {
		return fmt.Errorf("error asignando hostname: %w", err)
	}

	// 3. Configurar DNS dentro del rootfs antes del pivot_root
	_ = os.WriteFile(rootfs+"/etc/resolv.conf", []byte("nameserver 8.8.8.8\nnameserver 1.1.1.1\n"), 0644)

	// 4. Cambio de sistema de archivos raíz
	if err := PivotRoot(rootfs); err != nil {
		return fmt.Errorf("error en PivotRoot: %w", err)
	}

	// 5. Montar /proc independiente
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("error montando /proc: %w", err)
	}
	defer syscall.Unmount("/proc", 0)

	// 6. Montar /dev/pts para manejo correcto de terminales
	_ = os.MkdirAll("/dev/pts", 0755)
	_ = syscall.Mount("devpts", "/dev/pts", "devpts", 0, "")
	defer syscall.Unmount("/dev/pts", 0)

	// 7. Reemplazar proceso por el comando final
	binaryPath, err := exec.LookPath(command[0])
	if err != nil {
		return fmt.Errorf("comando no encontrado dentro del rootfs: %w", err)
	}

	return syscall.Exec(binaryPath, command, os.Environ())
}