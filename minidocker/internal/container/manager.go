package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"minidocker/internal/isolation"
	"minidocker/internal/network"
	"minidocker/pkg/decorators"
)

const defaultStateDir = "/var/lib/minidocker/containers"

type Manager struct {
	baseDir   string
	storage   StorageDriver
	network   NetworkDriver
	isolation IsolationDriver
}

// NewManager inyecta las dependencias del motor
func NewManager(customBaseDir ...string) *Manager {
	dir := defaultStateDir
	if len(customBaseDir) > 0 && customBaseDir[0] != "" {
		dir = customBaseDir[0]
	}
	return &Manager{
		baseDir:   dir,
		storage:   &DefaultStorageDriver{},
		network:   &DefaultNetworkDriver{},
		isolation: &DefaultIsolationDriver{},
	}
}

func generateID() string {
	bytes := make([]byte, 6)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (m *Manager) CreateContainer(image string, cmd []string, opts ...Option) (*Container, error) {
	var container *Container

	err := decorators.WithLogging("Creando contenedor", func() error {
		id := generateID()
		cfg := Config{
			ID:        id,
			Name:      id,
			Image:     image,
			Command:   cmd,
			BasePath:  m.baseDir,
			CreatedAt: time.Now(),
			Limits: isolation.CgroupLimits{
				MemoryLimitBytes: 100 * 1024 * 1024,
				PidsMax:          20,
			},
		}

		for _, opt := range opts {
			opt(&cfg)
		}

		if existing, _ := m.GetContainer(cfg.Name); existing != nil {
			return fmt.Errorf("ya existe un contenedor con el nombre [%s] (ID: %s)", cfg.Name, existing.Config.ID)
		}

		container = &Container{
			Config: cfg,
			State:  StateCreated,
		}

		return m.saveMetadata(container)
	})

	if err != nil {
		return nil, err
	}
	return container, nil
}

// StartContainer inicia un contenedor previamente creado o detenido
func (m *Manager) StartContainer(idOrName string, overrideCmd []string) error {
	c, err := m.GetContainer(idOrName)
	if err != nil {
		return err
	}

	if c.State == StateRunning && c.PID > 0 && syscall.Kill(c.PID, 0) == nil {
		return fmt.Errorf("el contenedor [%s] ya está en ejecución (PID: %d)", c.Config.Name, c.PID)
	}

	cmdToRun := c.Config.Command
	if len(overrideCmd) > 0 {
		cmdToRun = overrideCmd
	}
	c.Config.Command = cmdToRun

	// Aplicación del decorador al montaje de OverlayFS
	var mergedRootFS string
	err = decorators.WithLogging(fmt.Sprintf("Montando storage para [%s]", c.Config.Name), func() error {
		var errMount error
		mergedRootFS, errMount = m.storage.Prepare(c.Config.ID, c.Config.Image, m.baseDir)
		return errMount
	})
	if err != nil {
		return err
	}
	defer m.storage.Cleanup(c.Config.ID, m.baseDir)

	c.Config.RootFS = mergedRootFS
	c.State = StateRunning
	c.StartedAt = time.Now()
	_ = m.saveMetadata(c)

	// Iniciar DNS y precargar tablas
	gwIP := c.Config.GatewayIP
	if gwIP == "" {
		gwIP = "172.19.0.1"
	}
	network.StartEmbeddedDNS(gwIP)

	onReady := func(pid int, ip string) {
		c.PID = pid
		c.Config.IP = ip
		c.Config.StaticIP = ip
		_ = m.saveMetadata(c)
	}

	// Ejecución aislada decorada
	runErr := decorators.WithLogging(fmt.Sprintf("Ejecutando contenedor [%s]", c.Config.Name), func() error {
		return m.isolation.Run(context.Background(), c.Config, mergedRootFS, onReady)
	})

	c.StoppedAt = time.Now()
	c.PID = 0
	if runErr != nil {
		c.State = StateFailed
		_ = m.saveMetadata(c)
		return runErr
	}

	c.State = StateStopped
	c.ExitCode = 0
	_ = m.saveMetadata(c)
	return nil
}

func (m *Manager) RunContainer(c *Container) error {
	return m.StartContainer(c.Config.ID, nil)
}

func (m *Manager) StopContainer(idOrName string) error {
	c, err := m.GetContainer(idOrName)
	if err != nil {
		return err
	}

	if c.State != StateRunning {
		return fmt.Errorf("el contenedor [%s] no está en ejecución (estado: %s)", c.Config.Name, c.State)
	}

	return decorators.WithLogging(fmt.Sprintf("Deteniendo [%s]", c.Config.Name), func() error {
		if err := m.isolation.Stop(c.Config.ID, c.PID); err != nil {
			return err
		}

		c.State = StateStopped
		c.StoppedAt = time.Now()
		c.PID = 0
		return m.saveMetadata(c)
	})
}

func (m *Manager) DeleteContainer(idOrName string) error {
	c, err := m.GetContainer(idOrName)
	if err != nil {
		return err
	}

	if c.State == StateRunning && c.PID > 0 && syscall.Kill(c.PID, 0) == nil {
		return fmt.Errorf("el contenedor [%s] está en ejecución. Deténlo antes de eliminarlo", c.Config.Name)
	}

	return decorators.WithLogging(fmt.Sprintf("Eliminando [%s]", c.Config.Name), func() error {
		return m.storage.Delete(c.Config.ID, m.baseDir)
	})
}

func (m *Manager) saveMetadata(c *Container) error {
	dir := filepath.Join(m.baseDir, c.Config.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	configPath := filepath.Join(dir, "config.json")
	return os.WriteFile(configPath, data, 0644)
}

func (m *Manager) GetContainer(idOrName string) (*Container, error) {
	configPath := filepath.Join(m.baseDir, idOrName, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var c Container
		if err := json.Unmarshal(data, &c); err == nil {
			return &c, nil
		}
	}

	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("contenedor no encontrado: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidatePath := filepath.Join(m.baseDir, entry.Name(), "config.json")
		data, err := os.ReadFile(candidatePath)
		if err != nil {
			continue
		}

		var c Container
		if err := json.Unmarshal(data, &c); err == nil {
			if c.Config.Name == idOrName || c.Config.ID == idOrName {
				return &c, nil
			}
		}
	}

	return nil, fmt.Errorf("no se encontró ningún contenedor con ID o nombre [%s]", idOrName)
}

// ListContainers recorre los directorios de estado y devuelve todas las entidades
func (m *Manager) ListContainers() ([]*Container, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Container{}, nil
		}
		return nil, fmt.Errorf("error leyendo baseDir: %w", err)
	}

	var containers []*Container
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		c, err := m.GetContainer(entry.Name())
		if err != nil {
			continue
		}
		containers = append(containers, c)
	}
	return containers, nil
}
