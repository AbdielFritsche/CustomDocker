package network

import (
	"fmt"
	"net"
	"runtime"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// SetupContainerNetwork crea el par veth, mueve un extremo al PID y configura IP y rutas
func SetupContainerNetwork(containerPID int, containerIP string) error {
	// 1. Bloquear la goroutine actual a su hilo POSIX para que los cambios de netns no se mezclen
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	vethHostName := fmt.Sprintf("veth0_%d", containerPID)
	vethContName := fmt.Sprintf("veth1_%d", containerPID)

	// 2. Crear par de interfaces virtuales
	vethLinkAttrs := netlink.NewLinkAttrs()
	vethLinkAttrs.Name = vethHostName

	veth := &netlink.Veth{
		LinkAttrs: vethLinkAttrs,
		PeerName:  vethContName,
	}

	if err := netlink.LinkAdd(veth); err != nil {
		return fmt.Errorf("error creando par veth: %w", err)
	}

	// 3. Vincular extremo host al bridge
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

	// 4. Mover el extremo contenedor al namespace de red del subproceso
	vethCont, err := netlink.LinkByName(vethContName)
	if err != nil {
		return fmt.Errorf("error obteniendo extremo veth container: %w", err)
	}

	if err := netlink.LinkSetNsPid(vethCont, containerPID); err != nil {
		return fmt.Errorf("error moviendo veth al namespace de red: %w", err)
	}

	// 5. Guardar namespace del host
	hostNs, err := netns.Get()
	if err != nil {
		return fmt.Errorf("error obteniendo netns del host: %w", err)
	}
	defer hostNs.Close()

	// 6. Obtener namespace del contenedor
	nsHandle, err := netns.GetFromPid(containerPID)
	if err != nil {
		return fmt.Errorf("error obteniendo netns del contenedor: %w", err)
	}
	defer nsHandle.Close()

	// 7. Cambiar el hilo al netns del contenedor
	if err := netns.Set(nsHandle); err != nil {
		return fmt.Errorf("error configurando netns: %w", err)
	}
	defer netns.Set(hostNs) // Garantizar regreso al namespace del host

	// 8. Renombrar la interfaz a 'eth0' dentro del contenedor y levantarla
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

	// 9. Asignar IP al contenedor
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

	// 10. Levantar loopback (lo)
	if lo, err := netlink.LinkByName("lo"); err == nil {
		_ = netlink.LinkSetUp(lo)
	}

	// 11. Configurar Gateway por defecto apuntando al bridge
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

// CleanupNetwork elimina las interfaces virtuales del contenedor
func CleanupNetwork(containerPID int) {
	vethHostName := fmt.Sprintf("veth0_%d", containerPID)
	if link, err := netlink.LinkByName(vethHostName); err == nil {
		_ = netlink.LinkDel(link)
	}
}
