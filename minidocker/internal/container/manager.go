package container

import (
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
func (m *Manager) StartContainer(idOrName string, overrideCmd []string) error {
	c, err := m.GetContainer(idOrName)
	if err != nil {
		return err
	}

	if c.State == StateRunning {
		if c.PID > 0 && syscall.Kill(c.PID, 0) == nil {
			return fmt.Errorf("el contenedor [%s] ya está en ejecución (PID: %d)", c.Config.Name, c.PID)
		}
		c.State = StateStopped
		c.PID = 0
		_ = m.saveMetadata(c)
	}

	cmdToRun := c.Config.Command
	if len(overrideCmd) > 0 {
		cmdToRun = overrideCmd
	}

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

	bridgeName := "minibr0"
	bridgeIP := "172.19.0.1/16"
	subnetCIDR := "172.19.0.0/16"
	gatewayIP := "172.19.0.1"

	if c.Config.BridgeName != "" {
		bridgeName = c.Config.BridgeName
		bridgeIP = c.Config.BridgeIP
		subnetCIDR = c.Config.SubnetCIDR
		gatewayIP = c.Config.GatewayIP
	}

	onReady := func(pid int, ip string) {
		c.PID = pid
		c.Config.IP = ip
		c.Config.StaticIP = ip
		_ = m.saveMetadata(c)
	}

	// 1. Iniciar servidor DNS en el gateway de este contenedor
	network.StartEmbeddedDNS(gatewayIP)

	// 2. Precargar todos los contenedores existentes en la tabla global
	if entries, err := os.ReadDir(m.baseDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if other, err := m.GetContainer(entry.Name()); err == nil {
				targetIP := other.Config.StaticIP
				if targetIP == "" {
					targetIP = other.Config.IP
				}
				otherGateway := other.Config.GatewayIP
				if otherGateway == "" {
					otherGateway = "172.19.0.1" // gateway por defecto
				}
				if targetIP != "" {
					network.RegisterRecord(otherGateway, other.Config.Name, targetIP)
					network.RegisterRecord(otherGateway, other.Config.ID, targetIP)
				}
			}
		}
	}

	runErr := isolation.RunParent(
		c.Config.ID,
		c.Config.Name,
		mergedRootFS,
		c.Config.Limits,
		cmdToRun,
		c.Config.Env,
		hp,
		cp,
		bridgeName,
		bridgeIP,
		subnetCIDR,
		gatewayIP,
		c.Config.StaticIP,
		onReady,
	)

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

	cg := isolation.NewCgroupManager(c.Config.ID)
	_ = cg.KillAll()

	// 2. Si tenía un PID principal registrado, enviar señal al Process Group (-PID)
	if c.PID > 0 {
		_ = syscall.Kill(-c.PID, syscall.SIGTERM)
		_ = syscall.Kill(c.PID, syscall.SIGTERM)

		// Esperar hasta 2 segundos a que el proceso principal cierre
		stopped := false
		for i := 0; i < 4; i++ {
			time.Sleep(500 * time.Millisecond)
			if err := syscall.Kill(c.PID, 0); err != nil {
				stopped = true
				break
			}
		}

		// Si sigue existiendo, forzar con SIGKILL
		if !stopped {
			_ = syscall.Kill(-c.PID, syscall.SIGKILL)
			_ = syscall.Kill(c.PID, syscall.SIGKILL)
		}
	}

	// 3. Limpiar cgroup y estado
	_ = cg.Cleanup()
	c.State = StateStopped
	c.StoppedAt = time.Now()
	c.PID = 0
	_ = m.saveMetadata(c)

	return nil
}

func (m *Manager) DeleteContainer(idOrName string) error {
	c, err := m.GetContainer(idOrName)
	if err != nil {
		return err
	}

	// 1. Validar que no esté corriendo
	if c.State == StateRunning && c.PID > 0 && syscall.Kill(c.PID, 0) == nil {
		return fmt.Errorf("el contenedor [%s] está en ejecución. Deténlo antes de eliminarlo", c.Config.Name)
	}

	containerDir := filepath.Join(m.baseDir, c.Config.ID)
	mergedDir := filepath.Join(containerDir, "merged")

	// 2. Desmontar de forma segura la capa OverlayFS (MNT_DETACH desliga el montaje aunque esté ocupado)
	_ = syscall.Unmount(mergedDir, syscall.MNT_DETACH)

	// 3. Eliminar todo el árbol de directorios de manera limpia
	if err := os.RemoveAll(containerDir); err != nil {
		return fmt.Errorf("error eliminando almacenamiento del contenedor: %w", err)
	}

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

func (m *Manager) GetContainer(idOrName string) (*Container, error) {
	// 1. Intento directo por ID
	configPath := filepath.Join(m.baseDir, idOrName, "config.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var c Container
		if err := json.Unmarshal(data, &c); err == nil {
			return &c, nil
		}
	}

	// 2. Búsqueda por Nombre
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

func (m *Manager) ListContainerDirs() ([]string, error) {
	entries, err := os.ReadDir(m.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("error leyendo el directorio base %s: %w", m.baseDir, err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			containerDirPath := filepath.Join(m.baseDir, entry.Name())
			dirs = append(dirs, containerDirPath)
		}
	}
	return dirs, nil
}
