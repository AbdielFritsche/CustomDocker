package network

import (
	"os/exec"

	"github.com/vishvananda/netlink"
)

// DeleteBridge apaga, elimina el bridge y remueve las reglas NAT/Forwarding asociadas
func DeleteBridge(bridgeName, subnetCIDR string) error {
	if link, err := netlink.LinkByName(bridgeName); err == nil {
		_ = netlink.LinkSetDown(link)
		_ = netlink.LinkDel(link)
	}

	// Revertir FORWARD
	_ = exec.Command("iptables", "-D", "FORWARD", "-o", bridgeName, "-j", "ACCEPT").Run()
	_ = exec.Command("iptables", "-D", "FORWARD", "-i", bridgeName, "-j", "ACCEPT").Run()

	// Revertir NAT Masquerade
	if subnetCIDR != "" {
		_ = exec.Command("iptables", "-t", "nat", "-D", "POSTROUTING", "-s", subnetCIDR, "!", "-o", bridgeName, "-j", "MASQUERADE").Run()
	}
	return nil
}
