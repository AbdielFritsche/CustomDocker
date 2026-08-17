package main

import (
	"fmt"
	"os"

	"minidocker/internal/container"
	"minidocker/internal/isolation"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: sudo ./minidocker run <comando>")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		imagePath := "assets/alpine"
		userCommand := os.Args[2:]
		if len(userCommand) == 0 {
			userCommand = []string{"/bin/sh"}
		}

		// 1. Inicializar el gestor de contenedores
		mgr := container.NewManager()

		// 2. Crear entidad, generar ID único y persistir metadata inicial
		c, err := mgr.CreateContainer(
			imagePath,
			userCommand,
			container.WithMemoryLimit(100*1024*1024), // 100MB
			container.WithPidsMax(20),                // 20 procesos
		)
		if err != nil {
			fmt.Printf("Error creando contenedor: %v\n", err)
			os.Exit(1)
		}

		// 3. Ejecutar ciclo de vida (OverlayFS -> Namespaces -> Cgroups -> Cleanup)
		if err := mgr.RunContainer(c); err != nil {
			fmt.Printf("Error ejecutando contenedor: %v\n", err)
			os.Exit(1)
		}

	case "__init__":
		// Punto de entrada interno para el subproceso hijo aislado
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