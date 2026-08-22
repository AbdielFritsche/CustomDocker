package container

import "context"

// StorageDriver desacopla la preparación del sistema de archivos (OverlayFS)
type StorageDriver interface {
	Prepare(id, image, basePath string) (mergedPath string, err error)
	Cleanup(id, basePath string) error
	Delete(id, basePath string) error
}

// NetworkDriver desacopla la asignación de switches virtuales y veth
type NetworkDriver interface {
	SetupBridge(bridgeName, bridgeIP, subnetCIDR string) error
	SetupContainer(pid int, ip, bridgeName, gatewayIP string) error
	Cleanup(pid int)
}

// IsolationDriver desacopla la ejecución y gobierno con Namespaces y cgroups
type IsolationDriver interface {
	Run(ctx context.Context, cfg Config, rootFS string, onReady func(pid int, ip string)) error
	Stop(id string, pid int) error
}
