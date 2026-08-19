package compose

import (
	"fmt"

	"github.com/vishvananda/netlink"
)

func (e *Engine) Down(composePath string) error {
	cf, err := ParseComposeFile(composePath)
	if err != nil {
		return err
	}

	for serviceName := range cf.Services {
		fmt.Printf("[Compose] Limpiando servicio [%s]...\n", serviceName)
		_ = e.mgr.StopContainer(serviceName)
		_ = e.mgr.DeleteContainer(serviceName)
	}

	for netName := range cf.Networks {
		bridgeName := fmt.Sprintf("br_%s", netName)
		fmt.Printf("[Compose] Eliminando red [%s]...\n", bridgeName)
		if link, err := netlink.LinkByName(bridgeName); err == nil {
			_ = netlink.LinkSetDown(link)
			_ = netlink.LinkDel(link)
		}
	}

	return nil
}
