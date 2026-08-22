package compose

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/vishvananda/netlink"
)

func (e *Engine) Down(composePath string) error {
	cf, err := ParseComposeFile(composePath)
	if err != nil {
		return err
	}

	// 1. Detener y eliminar los contenedores del compose
	for serviceName := range cf.Services {
		fmt.Printf("[Compose] Limpiando servicio [%s]...\n", serviceName)
		_ = e.mgr.StopContainer(serviceName)
		_ = e.mgr.DeleteContainer(serviceName)
	}

	autoSubnetIndex := 20

	// 2. Limpiar redes y matar sus respectivos daemons DNS
	for netName, netDef := range cf.Networks {
		bridgeName := fmt.Sprintf("br_%s", netName)

		// Obtener la gateway IP de esta red
		var gwIP string
		if netDef.Subnet != "" {
			if netDef.Gateway != "" {
				gwIP = netDef.Gateway
			} else {
				var b1, b2, b3 int
				_, _ = fmt.Sscanf(netDef.Subnet, "%d.%d.%d", &b1, &b2, &b3)
				gwIP = fmt.Sprintf("%d.%d.%d.1", b1, b2, b3)
			}
		} else {
			gwIP = fmt.Sprintf("172.%d.0.1", autoSubnetIndex)
			autoSubnetIndex++
		}

		// Matar el daemon DNS independiente y limpiar sus archivos de estado
		pidPath := fmt.Sprintf("/var/run/minidocker/dns/%s.pid", gwIP)
		if data, err := os.ReadFile(pidPath); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
			_ = os.Remove(pidPath)
			_ = os.Remove(fmt.Sprintf("/var/run/minidocker/dns/%s.json", gwIP))
		}

		// Eliminar el bridge del host
		fmt.Printf("[Compose] Eliminando red [%s]...\n", bridgeName)
		if link, err := netlink.LinkByName(bridgeName); err == nil {
			_ = netlink.LinkSetDown(link)
			_ = netlink.LinkDel(link)
		}
	}

	return nil
}
