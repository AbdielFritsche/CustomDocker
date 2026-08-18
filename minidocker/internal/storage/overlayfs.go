package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

const baseStoragePath = "/var/lib/minidocker/containers"

type OverlayDriver struct {
	ContainerID string
	BasePath    string
	LowerDir    string
	UpperDir    string
	WorkDir     string
	MergedDir   string
}

func NewOverlayDriver(containerID, lowerDir string, basePath string) *OverlayDriver {
	if basePath == "" {
		basePath = baseStoragePath
	}
	base := filepath.Join(basePath, containerID)
	return &OverlayDriver{
		ContainerID: containerID,
		BasePath:    base,
		LowerDir:    lowerDir,
		UpperDir:    filepath.Join(base, "upper"),
		WorkDir:     filepath.Join(base, "work"),
		MergedDir:   filepath.Join(base, "merged"),
	}
}

func (o *OverlayDriver) Mount() (string, error) {
	for _, dir := range []string{o.UpperDir, o.WorkDir, o.MergedDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("error creando directorio %s: %w", dir, err)
		}
	}

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", o.LowerDir, o.UpperDir, o.WorkDir)
	if err := syscall.Mount("overlay", o.MergedDir, "overlay", 0, opts); err != nil {
		return "", fmt.Errorf("falló mount overlay: %w", err)
	}

	return o.MergedDir, nil
}

// Unmount desmonta la capa merged SIN borrar los directorios ni la metadata
func (o *OverlayDriver) Unmount() error {
	if err := syscall.Unmount(o.MergedDir, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("error desmontando %s: %w", o.MergedDir, err)
	}
	return nil
}
