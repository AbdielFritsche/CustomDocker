package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"minidocker/internal/isolation"
	"minidocker/internal/storage"
)

// generateID crea un ID aleatorio de 12 caracteres
func generateID() string {
	bytes := make([]byte, 6)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: sudo ./minidocker run <comando>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		absLower, err := filepath.Abs("assets/alpine")
		if err != nil {
			panic(err)
		}

		userCommand := os.Args[2:]
		if len(userCommand) == 0 {
			userCommand = []string{"/bin/sh"}
		}

		containerID := generateID()

		// 1. Inicializar y montar OverlayFS
		driver := storage.NewOverlayDriver(containerID, absLower)
		mergedRootFS, err := driver.Mount()
		if err != nil {
			fmt.Printf("Error montando storage: %v\n", err)
			os.Exit(1)
		}
		defer driver.Unmount() // Limpieza garantizada al finalizar

		// 2. Límites de recursos
		limits := isolation.CgroupLimits{
			MemoryLimitBytes: 100 * 1024 * 1024, // 100MB
			PidsMax:          20,
		}

		// 3. Ejecutar el contenedor usando mergedRootFS
		if err := isolation.RunParent(containerID, mergedRootFS, limits, userCommand); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

	case "__init__":
		rootfs := os.Args[2]
		userCommand := os.Args[3:]

		if err := isolation.RunChild(rootfs, userCommand); err != nil {
			fmt.Printf("Error en child: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Printf("Comando desconocido: %s\n", os.Args[1])
		os.Exit(1)
	}
}