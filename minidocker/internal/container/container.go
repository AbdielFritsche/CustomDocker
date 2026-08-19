package container

import (
	"fmt"
	"minidocker/internal/isolation"
	"time"
)

// State representa el estado actual del contenedor en su ciclo de vida
type State string

const (
	StateCreated State = "created"
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateFailed  State = "failed"
)

type PortMapping struct {
	HostPort      int `json:"host_port"`
	ContainerPort int `json:"container_port"`
}

// Config contiene la especificación base para crear un contenedor
type Config struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Image       string                 `json:"image"`
	Command     []string               `json:"command"`
	Env         []string               `json:"environment"`
	RootFS      string                 `json:"rootfs"`
	IP          string                 `json:"ip"`
	BasePath    string                 `json:"base_path"`
	Network     string                 `json:"network,omitempty"`
	BridgeName  string                 `json:"bridge_name,omitempty"`
	BridgeIP    string                 `json:"bridge_ip,omitempty"`
	SubnetCIDR  string                 `json:"subnet_cidr,omitempty"`
	GatewayIP   string                 `json:"gateway_ip,omitempty"`
	StaticIP    string                 `json:"static_ip,omitempty"`
	PortMapping *PortMapping           `json:"port_mapping,omitempty"`
	Limits      isolation.CgroupLimits `json:"limits"`
	CreatedAt   time.Time              `json:"created_at"`
}

// Container representa la entidad viva y persistida
type Container struct {
	Config    Config    `json:"config"`
	State     State     `json:"state"`
	PID       int       `json:"pid"`
	ExitCode  int       `json:"exit_code"`
	StartedAt time.Time `json:"started_at"`
	StoppedAt time.Time `json:"stopped_at"`
}

type Option func(*Config)

func WithMemoryLimit(bytes int64) Option {
	return func(c *Config) {
		c.Limits.MemoryLimitBytes = bytes
	}
}

func WithPidsMax(pids int64) Option {
	return func(c *Config) {
		c.Limits.PidsMax = pids
	}
}

func WithName(name string) Option {
	return func(c *Config) {
		c.Name = name
	}
}

func WithStaticIP(static_ip string) Option {
	return func(c *Config) {
		c.StaticIP = static_ip
	}
}

func WithNetwork(networkName string) Option {
	return func(c *Config) {
		c.Network = networkName
	}
}

func WithBasePath(path string) Option {
	return func(c *Config) {
		if path != "" {
			c.BasePath = path
		}
	}
}

func WithNetworkConfig(bridgeName, bridgeIP, subnetCIDR, gatewayIP string) Option {
	return func(c *Config) {
		c.BridgeName = bridgeName
		c.BridgeIP = bridgeIP
		c.SubnetCIDR = subnetCIDR
		c.GatewayIP = gatewayIP
	}
}

func WithPortMapping(mapping string) Option {
	return func(c *Config) {
		var hp, cp int
		if _, err := fmt.Sscanf(mapping, "%d:%d", &hp, &cp); err == nil {
			c.PortMapping = &PortMapping{
				HostPort:      hp,
				ContainerPort: cp,
			}
		}
	}
}

func WithEnv(env []string) Option {
	return func(c *Config) {
		c.Env = env
	}
}
