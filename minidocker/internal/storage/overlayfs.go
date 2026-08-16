package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const baseStoragePath = "/var/lib/minidocker/containers"

// OverlayDriver administra los puntos de montaje OverlayFS
type OverlayDriver struct {
	ContainerID string
	BasePath    string
	LowerDir    string
	UpperDir    string
	WorkDir     string
	MergedDir   string
}

// NewOverlayDriver inicializa las rutas para un contenedor
func NewOverlayDriver(containerID, lowerDir string) *OverlayDriver {
	base := filepath.Join(baseStoragePath, containerID)
	return &OverlayDriver{
		ContainerID: containerID,
		BasePath:    base,
		LowerDir:    lowerDir,
		UpperDir:    filepath.Join(base, "upper"),
		WorkDir:     filepath.Join(base, "work"),
		MergedDir:   filepath.Join(base, "merged"),
	}
}

// Mount crea los directorios necesarios y monta el sistema OverlayFS
func (o *OverlayDriver) Mount() (string, error) {
	// 1. Crear las carpetas de trabajo del contenedor
	for _, dir := range []string{o.UpperDir, o.WorkDir, o.MergedDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("error creando directorio %s: %w", dir, err)
		}
	}

	// 2. Construir los argumentos para la opción overlay
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", o.LowerDir, o.UpperDir, o.WorkDir)

	// 3. Syscall Mount tipo 'overlay'
	if err := syscall.Mount("overlay", o.MergedDir, "overlay", 0, opts); err != nil {
		return "", fmt.Errorf("falló mount overlay: %w", err)
	}

	return o.MergedDir, nil
}

// Unmount desmonta la capa merged y limpia los directorios temporales
func (o *OverlayDriver) Unmount() error {
	// Desmontar el directorio unificado
	if err := syscall.Unmount(o.MergedDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("error desmontando %s: %w", o.MergedDir, err)
	}

	// Eliminar el árbol de directorios del contenedor
	return os.RemoveAll(o.BasePath)
}