package compose

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ServiceConfig struct {
	Image       string          `yaml:"image"`
	Command     []string        `yaml:"command,omitempty"`
	Environment []string        `yaml:"environment,omitempty"`
	Ports       []string        `yaml:"ports,omitempty"`
	Networks    ServiceNetworks `yaml:"networks,omitempty"`
	MemoryMB    int64           `yaml:"memory,omitempty"`
	PidsMax     int64           `yaml:"pids_max,omitempty"`
}

type ComposeFile struct {
	Version  string                   `yaml:"version"`
	Services map[string]ServiceConfig `yaml:"services"`
	Networks map[string]NetworkDef    `yaml:"networks,omitempty"`
}

// NetworkDef define los datos de configuración de una red en la sección networks global
type NetworkDef struct {
	Subnet  string `yaml:"subnet,omitempty"`
	Gateway string `yaml:"gateway,omitempty"`
}

// ServiceNetworkConfig almacena la configuración de red asociada al servicio (como la IP)
type ServiceNetworkConfig struct {
	IPv4Address string `yaml:"ipv4_address,omitempty"`
}

// ServiceNetworks soporta tanto una lista ([]string) como un mapa (map[string]ServiceNetworkConfig)
type ServiceNetworks map[string]ServiceNetworkConfig

func (sn *ServiceNetworks) UnmarshalYAML(value *yaml.Node) error {
	*sn = make(map[string]ServiceNetworkConfig)
	if value.Kind == yaml.SequenceNode {
		var list []string
		if err := value.Decode(&list); err != nil {
			return err
		}
		for _, name := range list {
			(*sn)[name] = ServiceNetworkConfig{}
		}
		return nil
	}
	if value.Kind == yaml.MappingNode {
		type rawMap map[string]ServiceNetworkConfig
		var m rawMap
		if err := value.Decode(&m); err != nil {
			return err
		}
		*sn = ServiceNetworks(m)
		return nil
	}
	return nil
}

func ParseComposeFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error leyendo compose: %w", err)
	}
	return ParseComposeBytes(data)
}

func ParseComposeBytes(data []byte) (*ComposeFile, error) {
	var cf ComposeFile
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil, fmt.Errorf("error parseando YAML: %w", err)
	}
	return &cf, nil
}
