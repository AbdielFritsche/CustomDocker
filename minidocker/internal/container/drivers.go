package container

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"minidocker/internal/isolation"
	"minidocker/internal/network"
	"minidocker/internal/storage"
)

// DefaultStorageDriver implementa StorageDriver con OverlayFS
type DefaultStorageDriver struct{}

func (d *DefaultStorageDriver) Prepare(id, image, basePath string) (string, error) {
	lowerPath := image
	if stat, err := os.Stat(lowerPath); err != nil || !stat.IsDir() {
		downloadedPath, err := storage.PullImage(image)
		if err != nil {
			return "", err
		}
		lowerPath = downloadedPath
	}
	absLower, err := filepath.Abs(lowerPath)
	if err != nil {
		return "", err
	}
	driver := storage.NewOverlayDriver(id, absLower, basePath)
	return driver.Mount()
}

func (d *DefaultStorageDriver) Cleanup(id, basePath string) error {
	mergedDir := filepath.Join(basePath, id, "merged")
	return syscall.Unmount(mergedDir, syscall.MNT_DETACH)
}

func (d *DefaultStorageDriver) Delete(id, basePath string) error {
	_ = d.Cleanup(id, basePath)
	return os.RemoveAll(filepath.Join(basePath, id))
}

// DefaultNetworkDriver implementa NetworkDriver con Linux Bridges y veth
type DefaultNetworkDriver struct{}

func (n *DefaultNetworkDriver) SetupBridge(bridgeName, bridgeIP, subnetCIDR string) error {
	return network.SetupNamedBridge(bridgeName, bridgeIP, subnetCIDR)
}

func (n *DefaultNetworkDriver) SetupContainer(pid int, ip, bridgeName, gatewayIP string) error {
	return network.SetupContainerNetworkDynamic(pid, ip, bridgeName, gatewayIP)
}

func (n *DefaultNetworkDriver) Cleanup(pid int) {
	network.CleanupNetwork(pid)
}

// DefaultIsolationDriver implementa IsolationDriver usando Namespaces y Cgroups v2
type DefaultIsolationDriver struct{}

func (i *DefaultIsolationDriver) Run(ctx context.Context, cfg Config, rootFS string, onReady func(pid int, ip string)) error {
	var hp, cp int
	if cfg.PortMapping != nil {
		hp = cfg.PortMapping.HostPort
		cp = cfg.PortMapping.ContainerPort
	}
	return isolation.RunParent(
		cfg.ID,
		cfg.Name,
		rootFS,
		cfg.Limits,
		cfg.Command,
		cfg.Env,
		hp,
		cp,
		cfg.BridgeName,
		cfg.BridgeIP,
		cfg.SubnetCIDR,
		cfg.GatewayIP,
		cfg.StaticIP,
		onReady,
	)
}

// DefaultIsolationDriver gestiona la finalización limpia de procesos y cgroups
func (i *DefaultIsolationDriver) Stop(id string, pid int) error {
	cg := isolation.NewCgroupManager(id)
	_ = cg.KillAll()

	if pid > 0 {
		_ = syscall.Kill(-pid, syscall.SIGTERM)
		_ = syscall.Kill(pid, syscall.SIGTERM)

		// Esperar hasta 2 segundos a que el proceso cierre
		stopped := false
		for k := 0; k < 4; k++ {
			time.Sleep(500 * time.Millisecond)
			if err := syscall.Kill(pid, 0); err != nil {
				stopped = true
				break
			}
		}

		// Si no respondió, forzar con SIGKILL
		if !stopped {
			_ = syscall.Kill(-pid, syscall.SIGKILL)
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}

	return cg.Cleanup()
}
