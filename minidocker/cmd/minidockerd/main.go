package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"minidocker/internal/api"
	"minidocker/internal/container"
	"minidocker/internal/network"
)

const (
	defaultBridgeName = "minibr0"
	defaultBridgeIP   = "172.19.0.1/16"
	defaultSubnetCIDR = "172.19.0.0/16"
	defaultGatewayIP  = "172.19.0.1"
	socketPath        = "/var/run/minidocker.sock"
)

func main() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "minidockerd requiere privilegios de root (sudo)")
		os.Exit(1)
	}

	// 1. Configurar bridge nombrado por defecto
	if err := network.SetupNamedBridge(defaultBridgeName, defaultBridgeIP, defaultSubnetCIDR); err != nil {
		fmt.Fprintf(os.Stderr, "Error configurando bridge predeterminado %s: %v\n", defaultBridgeName, err)
		os.Exit(1)
	}

	// 2. Levantar el daemon DNS desacoplado para la subred base
	network.StartEmbeddedDNS(defaultGatewayIP)

	// 3. Inicializar Manager pasando la ruta base de estado
	mgr := container.NewManager("/var/lib/minidocker/containers")

	// 4. Instanciar y arrancar el servidor API sobre Unix Socket
	server := api.NewServer(socketPath, mgr)

	// Limpieza garantizada del socket al apagar con SIGINT/SIGTERM
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		_ = os.Remove(socketPath)
		os.Exit(0)
	}()

	fmt.Printf("[minidockerd] Daemon activo en %s...\n", socketPath)
	if err := server.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Fallo en daemon: %v\n", err)
		os.Exit(1)
	}
}
