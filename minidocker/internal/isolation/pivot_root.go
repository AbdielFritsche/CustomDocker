package isolation

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// PivotRoot intercambia de forma segura la raíz del proceso al newRoot
func PivotRoot(newRoot string) error {
	// 1. Convertir la raíz del namespace a privada para no afectar al host
	if err := syscall.Mount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		return fmt.Errorf("error haciendo mount private en /: %w", err)
	}

	// 2. Bind mount de newRoot sobre sí mismo
	if err := syscall.Mount(newRoot, newRoot, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("error haciendo bind mount en newRoot: %w", err)
	}

	putOld := filepath.Join(newRoot, ".oldroot")
	if err := os.MkdirAll(putOld, 0700); err != nil {
		return fmt.Errorf("error creando directorio .oldroot: %w", err)
	}

	// 3. Syscall PivotRoot
	if err := syscall.PivotRoot(newRoot, putOld); err != nil {
		return fmt.Errorf("falló syscall.PivotRoot: %w", err)
	}

	// 4. Cambiar directorio de trabajo a la nueva raíz
	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("error cambiando a /: %w", err)
	}

	// 5. Montar un procfs nuevo, propio del namespace del contenedor
	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil {
		return fmt.Errorf("error montando /proc: %w", err)
	}

	// 6. Desmontar el root viejo
	putOldUnmount := "/.oldroot"
	if err := syscall.Unmount(putOldUnmount, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("error desmontando oldroot: %w", err)
	}

	return os.Remove(putOldUnmount)
}
