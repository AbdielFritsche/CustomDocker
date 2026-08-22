package compose

import (
	"fmt"
	"minidocker/internal/container"
	"minidocker/internal/network"
)

type Engine struct {
	mgr *container.Manager
}

func NewEngine(mgr *container.Manager) *Engine {
	return &Engine{mgr: mgr}
}

type netInfo struct {
	bridgeName string
	gwIP       string
	gwCIDR     string
	subnetCIDR string
}

func (e *Engine) Up(composePath string) error {
	cf, err := ParseComposeFile(composePath)
	if err != nil {
		return err
	}

	networksMap := make(map[string]netInfo)
	autoSubnetIndex := 20

	// 1. Inicializar bridges y sus servidores DNS
	for netName, netDef := range cf.Networks {
		bridgeName := fmt.Sprintf("br_%s", netName)
		var gwIP, gwCIDR, subnetCIDR string

		if netDef.Subnet != "" {
			subnetCIDR = netDef.Subnet
			if netDef.Gateway != "" {
				gwIP = netDef.Gateway
			} else {
				var b1, b2, b3 int
				_, _ = fmt.Sscanf(subnetCIDR, "%d.%d.%d", &b1, &b2, &b3)
				gwIP = fmt.Sprintf("%d.%d.%d.1", b1, b2, b3)
			}
			gwCIDR = fmt.Sprintf("%s/24", gwIP)
		} else {
			gwIP = fmt.Sprintf("172.%d.0.1", autoSubnetIndex)
			gwCIDR = fmt.Sprintf("%s/16", gwIP)
			subnetCIDR = fmt.Sprintf("172.%d.0.0/16", autoSubnetIndex)
			autoSubnetIndex++
		}

		fmt.Printf("[Compose] Inicializando red aislada [%s] en %s (%s, GW: %s)...\n", netName, bridgeName, subnetCIDR, gwIP)
		if err := network.SetupNamedBridge(bridgeName, gwCIDR, subnetCIDR); err != nil {
			return err
		}

		// Levantar DNS para esta red
		network.StartEmbeddedDNS(gwIP)

		networksMap[netName] = netInfo{
			bridgeName: bridgeName,
			gwIP:       gwIP,
			gwCIDR:     gwCIDR,
			subnetCIDR: subnetCIDR,
		}
	}

	// 2. Crear los servicios registrados
	for serviceName, svc := range cf.Services {
		fmt.Printf("[Compose] Creando servicio [%s] (Imagen: %s)...\n", serviceName, svc.Image)

		cmd := svc.Command
		if len(cmd) == 0 {
			cmd = []string{"/bin/sh"}
		}

		memLimit := svc.MemoryMB
		if memLimit == 0 {
			memLimit = 100
		}

		pids := svc.PidsMax
		if pids == 0 {
			pids = 50
		}

		opts := []container.Option{
			container.WithName(serviceName),
			container.WithMemoryLimit(memLimit * 1024 * 1024),
			container.WithPidsMax(pids),
		}

		if len(svc.Ports) > 0 {
			opts = append(opts, container.WithPortMapping(svc.Ports[0]))
		}

		if len(svc.Environment) > 0 {
			opts = append(opts, container.WithEnv(svc.Environment))
		}

		for netName, netCfg := range svc.Networks {
			opts = append(opts, container.WithNetwork(netName))
			nInfo, exists := networksMap[netName]
			if exists {
				opts = append(opts, container.WithNetworkConfig(nInfo.bridgeName, nInfo.gwCIDR, nInfo.subnetCIDR, nInfo.gwIP))
			}
			if netCfg.IPv4Address != "" {
				opts = append(opts, container.WithStaticIP(netCfg.IPv4Address))
				if exists {
					network.RegisterRecord(nInfo.gwIP, serviceName, netCfg.IPv4Address)
				}
			}
			break
		}

		c, err := e.mgr.CreateContainer(svc.Image, cmd, opts...)
		if err != nil {
			return fmt.Errorf("error creando servicio %s: %w", serviceName, err)
		}

		fmt.Printf("[Compose] Contenedor [%s] listo para el servicio %s.\n", c.Config.ID, serviceName)
	}

	return nil
}
