package network

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

const (
	BridgeName = "minibr0"
	BridgeIP   = "172.19.0.1/16"
)

// SetupBridge crea y configura el switch virtual en el host
func SetupBridge() error {
	// 1. Validar si ya existe el bridge
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

	// 2. Asignar IP al bridge
	addr, err := netlink.ParseAddr(BridgeIP)
	if err != nil {
		return fmt.Errorf("error parseando IP del bridge: %w", err)
	}

	_ = netlink.AddrAdd(link, addr)

	// 3. Levantar la interfaz
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("error levantando bridge: %w", err)
	}

	// 4. Habilitar reenvío IP y NAT con iptables en el host
	_ = exec.Command("sysctl", "-w", "net.ipv4.ip_forward=1").Run()
	_ = exec.Command("iptables", "-t", "nat", "-A", "POSTROUTING", "-s", "172.19.0.0/16", "!", "-o", BridgeName, "-j", "MASQUERADE").Run()

	return nil
}

// SetupContainerNetwork crea el par veth, mueve un extremo al PID y configura IP y rutas
func SetupContainerNetwork(containerPID int, containerIP string) error {
	vethHostName := fmt.Sprintf("veth0_%d", containerPID)
	vethContName := fmt.Sprintf("veth1_%d", containerPID)

	// 1. Crear par de interfaces virtuales
	vethLinkAttrs := netlink.NewLinkAttrs()
	vethLinkAttrs.Name = vethHostName

	veth := &netlink.Veth{
		LinkAttrs: vethLinkAttrs,
		PeerName:  vethContName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("error creando par veth: %w", err)
	}

	// 2. Vincular el extremo host al bridge
	br, err := netlink.LinkByName(BridgeName)
	if err != nil {
		return fmt.Errorf("error encontrando bridge: %w", err)
	}

	vethHost, err := netlink.LinkByName(vethHostName)
	if err != nil {
		return fmt.Errorf("error obteniendo extremo veth host: %w", err)
	}

	if err := netlink.LinkSetMaster(vethHost, br); err != nil {
		return fmt.Errorf("error asociando veth al bridge: %w", err)
	}

	if err := netlink.LinkSetUp(vethHost); err != nil {
		return fmt.Errorf("error levantando veth host: %w", err)
	}

	// 3. Mover el extremo contenedor al namespace de red del subproceso
	vethCont, err := netlink.LinkByName(vethContName)
	if err != nil {
		return fmt.Errorf("error obteniendo extremo veth container: %w", err)
	}

	if err := netlink.LinkSetNsPid(vethCont, containerPID); err != nil {
		return fmt.Errorf("error moviendo veth al namespace de red: %w", err)
	}

	// 4. Entrar temporalmente al namespace de red del contenedor para configurar IP y rutas
	nsHandle, err := netns.GetFromPid(containerPID)
	if err != nil {
		return fmt.Errorf("error obteniendo netns del contenedor: %w", err)
	}
	defer nsHandle.Close()

	// Guardar namespace actual del host
	hostNs, err := netns.Get()
	if err != nil {
		return fmt.Errorf("error obteniendo netns del host: %w", err)
	}
	defer hostNs.Close()

	// Cambiar al netns del contenedor
	if err := netns.Set(nsHandle); err != nil {
		return fmt.Errorf("error configurando netns: %w", err)
	}
	defer netns.Set(hostNs) // Asegurar regreso al host

	// 5. Renombrar la interfaz a 'eth0' dentro del contenedor y levantarla
	cLink, err := netlink.LinkByName(vethContName)
	if err != nil {
		return fmt.Errorf("error localizando veth dentro de netns: %w", err)
	}

	if err := netlink.LinkSetName(cLink, "eth0"); err != nil {
		return fmt.Errorf("error renombrando interfaz a eth0: %w", err)
	}

	eth0, err := netlink.LinkByName("eth0")
	if err != nil {
		return fmt.Errorf("error obteniendo eth0: %w", err)
	}

	// Asignar IP al contenedor (ej. 172.19.0.2/16)
	ipAddr, err := netlink.ParseAddr(containerIP)
	if err != nil {
		return fmt.Errorf("error parseando IP de contenedor: %w", err)
	}

	if err := netlink.AddrAdd(eth0, ipAddr); err != nil {
		return fmt.Errorf("error asignando IP a eth0: %w", err)
	}

	if err := netlink.LinkSetUp(eth0); err != nil {
		return fmt.Errorf("error levantando eth0: %w", err)
	}

	// Levantar interfaz loopback (lo)
	lo, err := netlink.LinkByName("lo")
	if err == nil {
		_ = netlink.LinkSetUp(lo)
	}

	// Configurar Default Gateway apuntando al Bridge del Host (172.19.0.1)
	gw := net.ParseIP("172.19.0.1")
	route := &netlink.Route{
		Scope:     netlink.SCOPE_UNIVERSE,
		LinkIndex: eth0.Attrs().Index,
		Gw:        gw,
	}

	if err := netlink.RouteAdd(route); err != nil {
		return fmt.Errorf("error agregando default gateway: %w", err)
	}

	return nil
}

// CleanupNetwork elimina las interfaces virtuales creadas para un contenedor detenido
func CleanupNetwork(containerPID int) {
	vethHostName := fmt.Sprintf("veth0_%d", containerPID)
	if link, err := netlink.LinkByName(vethHostName); err == nil {
		_ = netlink.LinkDel(link)
	}
}