package network

import (
	"fmt"
	"os/exec"

	"github.com/vishvananda/netlink"
)

// DeleteBridge apaga, elimina el bridge y remueve las reglas NAT/Forwarding asociadas
func DeleteBridge(bridgeName string) error {
	link, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("la red/bridge [%s] no existe: %w", bridgeName, err)
	}

	// 1. Bajar y eliminar interfaz de Linux
	_ = netlink.LinkSetDown(link)
	if err := netlink.LinkDel(link); err != nil {
		return fmt.Errorf("error eliminando bridge %s: %w", bridgeName, err)
	}

	// 2. Limpieza de reglas iptables del bridge
	_ = exec.Command("iptables", "-D", "FORWARD", "-i", bridgeName, "-o", "minibr0", "-j", "DROP").Run()
	_ = exec.Command("iptables", "-D", "FORWARD", "-i", "minibr0", "-o", bridgeName, "-j", "DROP").Run()

	return nil
}
