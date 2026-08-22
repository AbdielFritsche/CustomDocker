package network

// NetworkManager actúa como el backend formal del NetworkDriver
type NetworkManager struct{}

func NewNetworkManager() *NetworkManager {
	return &NetworkManager{}
}

func (n *NetworkManager) SetupBridge(bridgeName, bridgeIP, subnetCIDR string) error {
	return SetupNamedBridge(bridgeName, bridgeIP, subnetCIDR)
}

func (n *NetworkManager) SetupContainer(pid int, ip, bridgeName, gatewayIP string) error {
	return SetupContainerNetworkDynamic(pid, ip, bridgeName, gatewayIP)
}

func (n *NetworkManager) Cleanup(pid int) {
	CleanupNetwork(pid)
}

func (n *NetworkManager) StartProxy(hostPort, containerPort int, containerIP string) (*TCPProxy, error) {
	return StartPortProxy(hostPort, containerPort, containerIP)
}
