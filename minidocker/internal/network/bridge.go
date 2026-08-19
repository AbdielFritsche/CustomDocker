package network

import (
	"fmt"
	"os/exec"

	"github.com/vishvananda/netlink"
)

// SetupNamedBridge crea un switch virtual con nombre y subred dinámicos
func SetupNamedBridge(bridgeName, bridgeIP, subnetCIDR string) error {
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName
	br := &netlink.Bridge{LinkAttrs: la}

	err := netlink.LinkAdd(br)
	if err != nil && err.Error() != "file exists" {
		return fmt.Errorf("error creando bridge %s: %w", bridgeName, err)
	}

	link, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("error localizando interfaz %s: %w", bridgeName, err)
	}

	addr, err := netlink.ParseAddr(bridgeIP)
	if err != nil {
		return fmt.Errorf("error parseando IP del bridge: %w", err)
	}

	// Evitar asignar múltiples IPs al mismo bridge
	addrs, err := netlink.AddrList(link, netlink.FAMILY_V4)
	alreadyAssigned := false
	if err == nil {
		for _, a := range addrs {
			if a.IPNet.String() == addr.IPNet.String() {
				alreadyAssigned = true
				break
			}
		}
	}

	if !alreadyAssigned {
		if err := netlink.AddrAdd(link, addr); err != nil && err.Error() != "file exists" {
			return fmt.Errorf("error asignando IP %s a %s: %w", bridgeIP, bridgeName, err)
		}
	}

	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("error levantando bridge %s: %w", bridgeName, err)
	}

	// 1. IP Forwarding global y soporte loopback
	_ = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	_ = exec.Command("sysctl", "-w", "net.ipv4.conf.all.route_localnet=1").Run()

	// 2. Permitir tráfico de FORWARD en el bridge
	_ = exec.Command("iptables", "-I", "FORWARD", "-o", bridgeName, "-j", "ACCEPT").Run()
	_ = exec.Command("iptables", "-I", "FORWARD", "-i", bridgeName, "-j", "ACCEPT").Run()

	// 3. Regla NAT (MASQUERADE) para salida a Internet
	_ = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", subnetCIDR, "!", "-o", bridgeName, "-j", "MASQUERADE").Run()

	return nil
}
