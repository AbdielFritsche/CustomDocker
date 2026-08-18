package isolation

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"

	"minidocker/internal/network"
)

// RunParent inicia el subproceso con aislamiento completo y reenvío de puertos
func RunParent(containerID, rootfs string, limits CgroupLimits, args []string, hostPort, containerPort int) error {
	if err := network.SetupBridge(); err != nil {
		return fmt.Errorf("error configurando bridge: %w", err)
	}

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
	cmd.ExtraFiles = []*os.File{pipeReader}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("error al iniciar el subproceso: %w", err)
	}

	pid := cmd.Process.Pid

	cg := NewCgroupManager(containerID)
	if err := cg.Apply(pid, limits); err != nil {
		_ = cmd.Process.Kill()
		_ = cg.Cleanup()
		return fmt.Errorf("error configurando cgroups: %w", err)
	}
	defer cg.Cleanup()

	// IP dinámica
	octet3 := (pid >> 8) & 0xFF
	octet4 := pid & 0xFF
	if octet4 == 0 || octet4 == 1 {
		octet4 = 2
	}
	rawIP := fmt.Sprintf("172.19.%d.%d", octet3, octet4)
	containerCIDR := fmt.Sprintf("%s/16", rawIP)

	if err := network.SetupContainerNetwork(pid, containerCIDR); err != nil {
		_ = cmd.Process.Kill()
		network.CleanupNetwork(pid)
		return fmt.Errorf("error configurando red del contenedor: %w", err)
	}
	defer network.CleanupNetwork(pid)

	// INICIAR PROXY: Esto abrirá el socket en WSL que activa wslrelay.exe en Windows
	if hostPort > 0 && containerPort > 0 {
		proxy, err := network.StartPortProxy(hostPort, containerPort, rawIP)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error iniciando proxy: %v\n", err)
		} else {
			fmt.Printf("[Red] Proxy activo en :%d -> %s:%d\n", hostPort, rawIP, containerPort)
			defer proxy.Close()
		}
	}

	_, _ = pipeWriter.Write([]byte("ready"))
	_ = pipeWriter.Close()

	return cmd.Wait()
}

// RunChild corre dentro de los nuevos namespaces aislados
func RunChild(rootfs string, command []string) error {
	syncPipe := os.NewFile(uintptr(3), "syncPipe")
	buf := make([]byte, 5)
	_, _ = io.ReadFull(syncPipe, buf)
	_ = syncPipe.Close()

	if err := syscall.Sethostname([]byte("minidocker")); err != nil {
		return fmt.Errorf("error asignando hostname: %w", err)
	}

	_ = os.WriteFile(rootfs+"/etc/resolv.conf", []byte("nameserver 8.8.8.8\nnameserver 1.1.1.1\n"), 0644)

	if err := PivotRoot(rootfs); err != nil {
		return fmt.Errorf("error en PivotRoot: %w", err)
	}

	// 1. Asegurar /proc
	_ = os.MkdirAll("/proc", 0755)
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("error montando /proc: %w", err)
	}
	defer syscall.Unmount("/proc", 0)

	// 2. Asegurar /dev/pts y permisos de terminal
	_ = os.MkdirAll("/dev/pts", 0755)
	_ = syscall.Mount("devpts", "/dev/pts", "devpts", 0, "")
	defer syscall.Unmount("/dev/pts", 0)

	// 3. Asegurar permisos correctos en directorios y dispositivos clave
	_ = os.MkdirAll("/tmp", 1777)
	_ = os.Chmod("/tmp", 01777)

	// Asegurar /dev/null funcional con permisos 0666
	if _, err := os.Stat("/dev/null"); err != nil {
		_ = os.WriteFile("/dev/null", []byte{}, 0666)
	}
	_ = os.Chmod("/dev/null", 0666)

	// 4. Buscar y ejecutar el binario solicitado
	binaryPath, err := exec.LookPath(command[0])
	if err != nil {
		return fmt.Errorf("comando no encontrado dentro del rootfs: %w", err)
	}

	return syscall.Exec(binaryPath, command, os.Environ())
}
