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

func NewManager(customBaseDir ...string) *Manager {
	dir := defaultStateDir
	if len(customBaseDir) > 0 && customBaseDir[0] != "" {
		dir = customBaseDir[0]
	}
	return &Manager{baseDir: dir}
}

func generateID() string {
	bytes := make([]byte, 6)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (m *Manager) CreateContainer(image string, cmd []string, opts ...Option) (*Container, error) {
	id := generateID()
	cfg := Config{
		ID:        id,
		Name:      id, // Por defecto el nombre es el ID
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

	// Validar que no exista otro contenedor con el mismo nombre
	if existing, _ := m.GetContainer(cfg.Name); existing != nil {
		return nil, fmt.Errorf("ya existe un contenedor con el nombre [%s] (ID: %s)", cfg.Name, existing.Config.ID)
	}

	container := &Container{
		Config: cfg,
		State:  StateCreated,
	}

	if err := m.saveMetadata(container); err != nil {
		return nil, fmt.Errorf("error guardando metadata: %w", err)
	}

	return container, nil
}

// StartContainer inicia un contenedor previamente creado o detenido
func (m *Manager) StartContainer(idOrName string) error {
	c, err := m.GetContainer(idOrName)
	if err != nil {
		return err
	}

	if c.State == StateRunning {
		return fmt.Errorf("el contenedor [%s] ya está en ejecución", c.Config.Name)
	}

	// Resolver imagen si es remota o local
	lowerPath := c.Config.Image
	if stat, err := os.Stat(lowerPath); err != nil || !stat.IsDir() {
		downloadedPath, err := storage.PullImage(c.Config.Image)
		if err != nil {
			return fmt.Errorf("no se pudo obtener la imagen [%s]: %w", c.Config.Image, err)
		}
		lowerPath = downloadedPath
	}

	absLower, err := filepath.Abs(lowerPath)
	if err != nil {
		return fmt.Errorf("ruta de imagen inválida: %w", err)
	}

	driver := storage.NewOverlayDriver(c.Config.ID, absLower, c.Config.BasePath)
	mergedRootFS, err := driver.Mount()
	if err != nil {
		return fmt.Errorf("error montando OverlayFS: %w", err)
	}
	defer driver.Unmount()

	c.Config.RootFS = mergedRootFS
	c.State = StateRunning
	c.StartedAt = time.Now()
	_ = m.saveMetadata(c)

	var hp, cp int
	if c.Config.PortMapping != nil {
		hp = c.Config.PortMapping.HostPort
		cp = c.Config.PortMapping.ContainerPort
	}

	err = isolation.RunParent(c.Config.ID, mergedRootFS, c.Config.Limits, c.Config.Command, hp, cp)

	c.StoppedAt = time.Now()
	c.PID = 0
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

func (m *Manager) DeleteContainer(idOrName string) error {
	c, err := m.GetContainer(idOrName)
	if err != nil {
		return err
	}

	if c.State == StateRunning {
		return fmt.Errorf("el contenedor [%s] está en ejecución. Deténlo antes de eliminarlo", c.Config.Name)
	}

	// Se borra usando su ID real de disco
	return os.RemoveAll(filepath.Join(c.Config.BasePath, c.Config.ID))
}

func (m *Manager) RunContainer(c *Container) error {
	return m.StartContainer(c.Config.ID)
}

func (m *Manager) saveMetadata(c *Container) error {
	dir := filepath.Join(c.Config.BasePath, c.Config.ID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}

func (m *Manager) GetContainer(idOrName string) (*Container, error) {
	// 1. Intento directo por ID
	configPath := filepath.Join(m.baseDir, idOrName, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var c Container
		if err := json.Unmarshal(data, &c); err == nil {
			return &c, nil
		}
	}

	// 2. Búsqueda por Nombre si el acceso directo falló
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		return nil, fmt.Errorf("contenedor [%s] no encontrado: %w", idOrName, err)
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
