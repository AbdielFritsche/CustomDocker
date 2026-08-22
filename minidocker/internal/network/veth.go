package network

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// SetupContainerNetworkDynamic conecta el namespace de red del contenedor al bridge
func SetupContainerNetworkDynamic(containerPID int, containerIP, bridgeName, gatewayIP string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	vethHostName := fmt.Sprintf("veth0_%d", containerPID)
	vethContName := fmt.Sprintf("veth1_%d", containerPID)

	// 1. Crear par veth
	vethLinkAttrs := netlink.NewLinkAttrs()
	vethLinkAttrs.Name = vethHostName

	veth := &netlink.Veth{
		LinkAttrs: vethLinkAttrs,
		PeerName:  vethContName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("error creando par veth: %w", err)
	}

	// 2. Asociar al bridge en el host
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return fmt.Errorf("error localizando bridge %s: %w", bridgeName, err)
	}

	vethHost, err := netlink.LinkByName(vethHostName)
	if err != nil {
		return fmt.Errorf("error obteniendo %s: %w", vethHostName, err)
	}

	if err := netlink.LinkSetMaster(vethHost, br); err != nil {
		return fmt.Errorf("error asociando %s al bridge %s: %w", vethHostName, bridgeName, err)
	}

	if err := netlink.LinkSetUp(vethHost); err != nil {
		return fmt.Errorf("error levantando %s: %w", vethHostName, err)
	}

	_ = netlink.LinkSetUp(br)

	// Desactivar reverse path filtering para evitar 'no route to host'
	_ = exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.rp_filter=0", vethHostName)).Run()
	_ = exec.Command("sysctl", "-w", fmt.Sprintf("net.ipv4.conf.%s.rp_filter=0", bridgeName)).Run()

	// 3. Mover extremo par al namespace del contenedor
	vethCont, err := netlink.LinkByName(vethContName)
	if err != nil {
		return fmt.Errorf("error obteniendo %s: %w", vethContName, err)
	}

	if err := netlink.LinkSetNsPid(vethCont, containerPID); err != nil {
		return fmt.Errorf("error moviendo veth al netns: %w", err)
	}

	// 4. Cambiar al netns del contenedor
	hostNs, err := netns.Get()
	if err != nil {
		return err
	}
	defer hostNs.Close()

	nsHandle, err := netns.GetFromPid(containerPID)
	if err != nil {
		return err
	}
	defer nsHandle.Close()

	if err := netns.Set(nsHandle); err != nil {
		return err
	}
	defer netns.Set(hostNs)

	// 5. Renombrar y levantar eth0 dentro del contenedor
	cLink, err := netlink.LinkByName(vethContName)
	if err != nil {
		return err
	}

	if err := netlink.LinkSetName(cLink, "eth0"); err != nil {
		return err
	}

	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return err
	}

	ipAddr, err := netlink.ParseAddr(containerIP)
	if err != nil {
		return fmt.Errorf("error parseando containerIP [%s]: %w", containerIP, err)
	}

	if err := netlink.AddrAdd(eth0, ipAddr); err != nil {
		return fmt.Errorf("error asignando IP %s a eth0: %w", containerIP, err)
	}

	if err := netlink.LinkSetUp(eth0); err != nil {
		return fmt.Errorf("error levantando eth0: %w", err)
	}

	if lo, err := netlink.LinkByName("lo"); err == nil {
		_ = netlink.LinkSetUp(lo)
	}

	// 6. Configurar ruta local de enlace y Default Gateway
	_, ipNet, _ := net.ParseCIDR(containerIP)
	linkRoute := &netlink.Route{
		Scope:     netlink.SCOPE_LINK,
		LinkIndex: eth0.Attrs().Index,
		Dst:       ipNet,
	}
	_ = netlink.RouteAdd(linkRoute)

	gw := net.ParseIP(gatewayIP)
	defaultRoute := &netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: eth0.Attrs().Index,
		Gw:        gw,
	}

	if err := netlink.RouteAdd(defaultRoute); err != nil {
		return fmt.Errorf("error asignando default gateway %s: %w", gatewayIP, err)
	}

	return nil
}

// CleanupNetwork remueve las interfaces veth del host
func CleanupNetwork(containerPID int) {
	vethHostName := fmt.Sprintf("veth0_%d", containerPID)
	if link, err := netlink.LinkByName(vethHostName); err == nil {
		_ = netlink.LinkDel(link)
	}
}
