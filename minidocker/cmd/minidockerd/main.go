package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	"minidocker/internal/api"
	"minidocker/internal/container"
	"minidocker/internal/isolation"
	"minidocker/internal/network"
)

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "__dnsd__" {
		gatewayIP := os.Args[2]
		if err := network.RunDNSDaemon(gatewayIP); err != nil {
			os.Exit(1)
		}
		return
	}
	if len(os.Args) >= 3 && os.Args[1] == "__init__" {
		rootfs := os.Args[2]
		userCommand := os.Args[3:]
		if err := isolation.RunChild(rootfs, userCommand); err != nil {
			fmt.Fprintf(os.Stderr, "Error en child: %v\n", err)
			os.Exit(1)
		}
		return
	}

	socketPath := flag.String("socket", api.DefaultSocketPath, "Ruta del socket UNIX de escucha")
	socketGroup := flag.String("socket-group", "minidocker", "Grupo de sistema dueño del socket (vacío para desactivar el chown)")
	dataRoot := flag.String("data-root", "/var/lib/minidocker/containers", "Ruta base de almacenamiento de contenedores")
	flag.Parse()

	if os.Geteuid() != 0 {
		log.Fatal("minidockerd requiere privilegios de root (namespaces, cgroups, netlink). Ejecuta con sudo.")
	}

	mgr := container.NewManager(*dataRoot)
	corrected, err := mgr.Reconcile()
	if err != nil {
		log.Fatalf("[minidockerd] error en reconciliacion de arranque: %v", err)
	}
	if len(corrected) > 0 {
		log.Printf("[minidockerd] reconciliacion: %d contenedor(es) corregidos(s) a 'stopped': %v", len(corrected), corrected)
	} else {
		log.Printf("[minidockerd] reconciliacion: sin inconsistencias")
	}

	srv := api.NewServer(mgr, *socketPath, *socketGroup)

	var shuttingDown atomic.Bool
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("[minidockerd] señal de apagado recibida, cerrando socket...")
		shuttingDown.Store(true)
		_ = srv.Close()
	}()

	log.Printf("[minidockerd] data-root: %s", *dataRoot)
	if err := srv.ListenAndServe(); err != nil && !shuttingDown.Load() {
		log.Fatalf("[minidockerd] error fatal: %v", err)
	}
	log.Println("[minidockerd] apagado limpio.")
}
