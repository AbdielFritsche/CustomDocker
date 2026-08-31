package isolation

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"minidocker/internal/network"

	"github.com/creack/pty"
)

type ControlEvent struct {
	Type string // "resize" | "kill"
	Cols uint16
	Rows uint16
}

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
	containerID, containerName, rootfs string,
	limits CgroupLimits,
	args []string,
	customEnv []string,
	hostPort, containerPort int,
	bridgeName, bridgeIP, subnetCIDR, gatewayIP, requestedIP string,
	inStream io.Reader,
	outStream io.Writer,
	errStream io.Writer,
	controlChan <-chan ControlEvent,
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

	network.StartEmbeddedDNS(gatewayIP)

	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("error creando pipe de sincronización: %w", err)
	}
	defer pipeReader.Close()

	childArgs := append([]string{"__init__", rootfs}, args...)
	cmd := exec.Command("/proc/self/exe", childArgs...)

	// Combinar variables de entorno de forma limpia
	cmd.Env = mergeEnviron(os.Environ(), append(customEnv, fmt.Sprintf("_MINIDOCKER_GATEWAY=%s", gatewayIP)))
	cmd.ExtraFiles = []*os.File{pipeReader}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWNS |
			syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWNET,
	}

	var ptmx *os.File
	usePTY := inStream != nil

	if usePTY {
		var ptyErr error
		ptmx, ptyErr = pty.Start(cmd)
		if ptyErr != nil {
			return fmt.Errorf("error iniciando proceso en PTY: %w", ptyErr)
		}
		defer func() { _ = ptmx.Close() }()

		go func() {
			_, _ = io.Copy(ptmx, inStream)
		}()

		if outStream != nil {
			go func() {
				_, _ = io.Copy(outStream, ptmx)
			}()
		}
	} else {
		cmd.Stdin = os.Stdin
		if outStream != nil {
			cmd.Stdout = outStream
		} else {
			cmd.Stdout = os.Stdout
		}

		if errStream != nil {
			cmd.Stderr = errStream
		} else {
			cmd.Stderr = os.Stderr
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("error al iniciar el subproceso: %w", err)
		}
	}
	pid := cmd.Process.Pid

	if controlChan != nil {
		go func() {
			for ev := range controlChan {
				switch ev.Type {
				case "resize":
					if usePTY && ptmx != nil && ev.Cols > 0 && ev.Rows > 0 {
						_ = pty.Setsize(ptmx, &pty.Winsize{Cols: ev.Cols, Rows: ev.Rows})
					}
				case "kill":
					_ = syscall.Kill(-pid, syscall.SIGKILL)
					_ = syscall.Kill(pid, syscall.SIGKILL)
				}
			}
		}()
	}

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
		return fmt.Errorf("error configurando red en bridge %s: %w", bridgeName, err)
	}
	defer network.CleanupNetwork(pid)

	network.RegisterRecord(gatewayIP, containerID, rawIP)
	network.RegisterRecord(gatewayIP, containerName, rawIP)
	defer network.UnregisterRecord(gatewayIP, containerID)
	defer network.UnregisterRecord(gatewayIP, containerName)

	if hostPort > 0 && containerPort > 0 {
		proxy, err := network.StartPortProxy(hostPort, containerPort, rawIP)
		if err == nil {
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
	if _, err := io.ReadFull(syncPipe, buf); err != nil {
		return fmt.Errorf("error leyendo pipe de sincronización: %w", err)
	}
	_ = syncPipe.Close()

	if err := syscall.Sethostname([]byte("minidocker")); err != nil {
		return fmt.Errorf("error asignando hostname: %w", err)
	}

	// Inyectar resolv.conf apuntando al gateway si existe en Env
	gwIP := os.Getenv("_MINIDOCKER_GATEWAY")
	if gwIP == "" {
		gwIP = "8.8.8.8"
	}
	_ = os.WriteFile(rootfs+"/etc/resolv.conf", fmt.Appendf(nil, "nameserver %s\nnameserver 8.8.8.8\n", gwIP), 0644)

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
