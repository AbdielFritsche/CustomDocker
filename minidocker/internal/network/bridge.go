package network

import (
	"fmt"
	"os/exec"

	"github.com/vishvananda/netlink"
)

const (
	BridgeName = "minibr0"
	BridgeIP   = "172.19.0.1/16"
)

// SetupBridge crea y configura el switch virtual en el host
func SetupBridge() error {
	la := netlink.NewLinkAttrs()
	la.Name = BridgeName
	br := &netlink.Bridge{LinkAttrs: la}

	err := netlink.LinkAdd(br)
	if err != nil && err.Error() != "file exists" {
		return fmt.Errorf("error creando bridge: %w", err)
	}

	link, err := netlink.LinkByName(BridgeName)
	if err != nil {
		return fmt.Errorf("error obteniendo interfaz bridge: %w", err)
	}

	addr, err := netlink.ParseAddr(BridgeIP)
	if err != nil {
		return fmt.Errorf("error parseando IP del bridge: %w", err)
	}

	_ = netlink.AddrAdd(link, addr)

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("error levantando bridge: %w", err)
	}

	// Habilitar IP Forwarding global y enrutamiento en loopback
	_ = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	_ = exec.Command("sysctl", "-w", "net.ipv4.conf.all.route_localnet=1").Run()
	_ = exec.Command("sysctl", "-w", "net.ipv4.conf.default.route_localnet=1").Run()

	// Regla general de salida NAT (MASQUERADE)
	_ = exec.Command("iptables", "-t", "nat", "-C", "POSTROUTING", "-s", "172.19.0.0/16", "!", "-o", BridgeName, "-j", "MASQUERADE").Run()
	if err := exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "172.19.0.0/16", "!", "-o", BridgeName, "-j", "MASQUERADE").Run(); err != nil {
		// Ignorar si ya existía
	}

	return nil
}
