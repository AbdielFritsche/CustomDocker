package container

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"minidocker/internal/isolation"
	"minidocker/internal/storage"
)

const defaultStateDir = "/var/lib/minidocker/containers"

type Manager struct {
	baseDir string
}

func NewManager() *Manager {
	return &Manager {
		baseDir: defaultStateDir,
	}
}


func generateID() string {
	bytes := make([]byte, 6)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (m *Manager) CreateContainer(image string, cmd []string, opts ...Option) (*Container, error) {
	id := generateID()
	cfg := Config {
		ID: id,
		Name: id,
		Image: image,
		Command: cmd,
		CreatedAt: time.Now(),
		Limits: isolation.CgroupLimits{
			MemoryLimitBytes: 100 * 1024 * 1024,
			PidsMax: 20,
		},
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	container := &Container {
		Config: cfg,
		State: StateCreated,
	}

	if err := m.saveMetadata(container); err != nil {
		return nil, fmt.Errorf("Error guardando metadata: %w", err)
	}

	return container,nil
}

func (m *Manager) RunContainer(c *Container) error {
	absLower, err := filepath.Abs(c.Config.Image)
	if err != nil {
		return fmt.Errorf("Ruta de imagen invalida: %w", err)
	}

	driver := storage.NewOverlayDriver(c.Config.ID, absLower)
	mergedRootFS, err := driver.Mount()
	if err != nil {
		return fmt.Errorf("Error montando OverlayFS: %w", err)
	}
	defer driver.Unmount()

	c.Config.RootFS = mergedRootFS
	c.State = StateRunning
	c.StartedAt = time.Now()
	_ = m.saveMetadata(c)

	err = isolation.RunParent(c.Config.ID, mergedRootFS, c.Config.Limits, c.Config.Command)

	c.StoppedAt = time.Now()
	if err != nil {
		c.State = StateFailed
		_ = m.saveMetadata(c)
		return err
	}

	c.State = StateStopped
	c.ExitCode = 0
	_ = m.saveMetadata(c)
	return nil
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

func (m *Manager) GetContainer(id string) (*Container, error) {
	configPath := filepath.Join(m.baseDir, id, "config.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("contenedor no encontrado: %w", err)
	}

	var c Container
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("error deserializando metadata: %w", err)
	}

	return &c, nil
}

// DeleteContainer valida el estado y elimina permanentemente los datos del contenedor
func (m *Manager) DeleteContainer(id string) error {
	c, err := m.GetContainer(id)
	if err != nil {
		return err
	}

	// Impedir el borrado si el contenedor sigue activo
	if c.State == StateRunning {
		return fmt.Errorf("el contenedor [%s] está en ejecución. Deténlo antes de eliminarlo", id)
	}

	containerDir := filepath.Join(m.baseDir, id)
	return os.RemoveAll(containerDir)
}