package isolation

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"minidocker/internal/network"
)

func mergeEnviron(baseEnv, customEnv []string) []string {
	envMap := make(map[string]string)
	for _, env := range baseEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for _, env := range customEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	var result []string
	for k, v := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func RunParent(
	containerID, rootfs string,
	limits CgroupLimits,
	args []string,
	customEnv []string,
	hostPort, containerPort int,
	bridgeName, bridgeIP, subnetCIDR, gatewayIP, requestedIP string,
	onReady func(pid int, ip string),
) error {
	if bridgeName == "" {
		bridgeName = "minibr0"
		bridgeIP = "172.19.0.1/16"
		subnetCIDR = "172.19.0.0/16"
		gatewayIP = "172.19.0.1"
	}

	if err := network.SetupNamedBridge(bridgeName, bridgeIP, subnetCIDR); err != nil {
		return fmt.Errorf("error configurando bridge %s: %w", bridgeName, err)
	}

	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("error creando pipe de sincronización: %w", err)
	}
	defer pipeReader.Close()

	childArgs := append([]string{"__init__", rootfs}, args...)
	cmd := exec.Command("/proc/self/exe", childArgs...)

	// Combinar variables de entorno de forma limpia
	cmd.Env = mergeEnviron(os.Environ(), customEnv)

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

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				_ = syscall.Kill(pid, sig.(syscall.Signal))
			}
		}
	}()
	defer signal.Stop(sigChan)

	cg := NewCgroupManager(containerID)
	if err := cg.Apply(pid, limits); err != nil {
		_ = cmd.Process.Kill()
		_ = cg.Cleanup()
		return fmt.Errorf("error configurando cgroups: %w", err)
	}
	defer cg.Cleanup()

	mask := "/16"
	if idx := strings.Index(subnetCIDR, "/"); idx != -1 {
		mask = subnetCIDR[idx:]
	}

	var rawIP string
	if requestedIP != "" {
		rawIP = requestedIP
	} else {
		var b1, b2 int
		_, _ = fmt.Sscanf(gatewayIP, "%d.%d", &b1, &b2)
		octet3 := (pid >> 8) & 0xFF
		octet4 := pid & 0xFF
		if octet4 == 0 || octet4 == 1 {
			octet4 = 2
		}
		rawIP = fmt.Sprintf("%d.%d.%d.%d", b1, b2, octet3, octet4)
	}

	containerCIDR := fmt.Sprintf("%s%s", rawIP, mask)

	if err := network.SetupContainerNetworkDynamic(pid, containerCIDR, bridgeName, gatewayIP); err != nil {
		_ = cmd.Process.Kill()
		network.CleanupNetwork(pid)
		return fmt.Errorf("error configurando red del contenedor en bridge %s: %w", bridgeName, err)
	}
	defer network.CleanupNetwork(pid)

	if hostPort > 0 && containerPort > 0 {
		proxy, err := network.StartPortProxy(hostPort, containerPort, rawIP)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Advertencia iniciando proxy: %v\n", err)
		} else {
			fmt.Printf("[Red] Proxy activo en :%d -> %s:%d\n", hostPort, rawIP, containerPort)
			defer proxy.Close()
		}
	}

	if onReady != nil {
		onReady(pid, rawIP)
	}

	_, _ = pipeWriter.Write([]byte("ready"))
	_ = pipeWriter.Close()

	return cmd.Wait()
}

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

	_ = os.MkdirAll("/proc", 0755)
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("error montando /proc: %w", err)
	}
	defer syscall.Unmount("/proc", 0)

	_ = os.MkdirAll("/dev/pts", 0755)
	_ = syscall.Mount("devpts", "/dev/pts", "devpts", 0, "")
	defer syscall.Unmount("/dev/pts", 0)

	_ = os.MkdirAll("/tmp", 1777)
	_ = os.Chmod("/tmp", 01777)

	if _, err := os.Stat("/dev/null"); err != nil {
		_ = os.WriteFile("/dev/null", []byte{}, 0666)
	}
	_ = os.Chmod("/dev/null", 0666)

	binaryPath, err := exec.LookPath(command[0])
	if err != nil {
		return fmt.Errorf("comando [%s] no encontrado dentro del rootfs: %w", command[0], err)
	}

	return syscall.Exec(binaryPath, command, os.Environ())
}
