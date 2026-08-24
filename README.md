# Vessel Engine

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" />
  <img src="https://img.shields.io/badge/Linux_Kernel-Syscalls-FCC624?style=for-the-badge&logo=linux&logoColor=black" alt="Linux Kernel" />
  <img src="https://img.shields.io/badge/cgroups-v2-informational?style=for-the-badge&logo=linux-foundation&logoColor=white" alt="cgroups v2" />
  <img src="https://img.shields.io/badge/OverlayFS-Storage-lightgrey?style=for-the-badge" alt="OverlayFS" />
  <img src="https://img.shields.io/badge/Networking-Bridge%20%7C%20veth%20%7C%20NAT-orange?style=for-the-badge" alt="Networking" />
  <img src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" alt="License MIT" />
</p>

Vessel es un runtime y orquestador de contenedores de bajo nivel implementado en Go sin dependencias de daemons externos ni herramientas como Docker o containerd. Interactúa directamente con las primitivas del kernel de Linux para ofrecer aislamiento de procesos (`namespaces`), gobernanza de hardware (`cgroups v2`), sistemas de archivos Copy-on-Write (`OverlayFS`) y redes virtuales dedicadas (`veth` + `bridges`).

> **ADVERTENCIA:** Este proyecto ejecuta llamadas al sistema privilegiadas (`sudo`). No está auditado para producción; se recomienda su ejecución en entornos de desarrollo aislados como WSL 2 o máquinas virtuales Linux.

---

## Características Principales

* **Aislamiento por Namespaces:** `PID` (PID 1 aislado), `NET`, `UTS`, `IPC` y `Mount` con `pivot_root`.
* **Gobernanza de Recursos:** Control de límites de memoria (`memory.max`) y mitigación de fork bombs (`pids.max`) mediante cgroups v2.
* **Almacenamiento Copy-on-Write:** Capas inmutables (`lowerdir`) combinadas con capas de escritura efímeras (`upperdir`) en OverlayFS.
* **Redes Virtuales:** Asignación dinámica de IPs, Linux Bridges dedicados, interfaces virtuales `veth` y proxy de reenvío de puertos TCP.
* **Orquestador Compose:** Motor declarativo para desplegar stacks de servicios interconectados (ej. bases de datos MySQL) mediante `compose.yaml`.
* **Interacción Dinámica:** Inyección de procesos en caliente (`exec`) mediante `unix.Setns` y streaming de logs estructurados (`logs -f`).

---

## Inicio Rápido (Quickstart)

```bash
# 1. Clonar y compilar el binario
git clone [https://github.com/TuUsuario/vessel.git](https://github.com/TuUsuario/vessel.git) && cd vessel
go build -o vessel cmd/vessel/main.go

# 2. Descargar rootfs base (Alpine Linux)
mkdir -p assets/alpine
curl -sL [https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz](https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz) | tar -xz -C assets/alpine

# 3. Iniciar un contenedor interactivo con límites de recursos y puertos
sudo ./vessel run --memory 256m --pids-max 50 -p 8080:80 /bin/sh
```

---

## Orquestación con Compose (Ejemplo MySQL)

```compose.yaml
services:
  database:
    image: "assets/mysql-rootfs"
    ports:
      - "3306:3306"
    memory: 1024
    pids_max: 150
```

```bash
# Desplegar el stack
sudo ./vessel compose up -f compose.yaml

# Detener y liberar recursos
sudo ./vessel compose down -f compose.yaml
```

---

## Comparativa Técnica

| Característica | Vessel | Docker | runc / Podman |
| :--- | :--- | :--- | :--- |
| **Arquitectura** | Daemonless / Modular | Cliente-Servidor (`dockerd`) | Daemonless (OCI Spec) |
| **Aislamiento** | Syscalls directas + cgroups v2 | libcontainerd + namespaces | libcontainer / cgroups |
| **Filesystem** | OverlayFS nativo en Go | Storage Drivers (Overlay2, VFS) | Bundles OCI |
| **Networking** | Linux Bridge + `veth` + Proxy TCP | Docker Bridge / CNI Plugins | Netavark / CNI |
| **Dependencias** | **Cero dependencias externas**| Requiere suite de daemons | Requiere herramientas OCI |

---

## Documentación Detallada (Wiki)

Para especificaciones internas, tablas de flags y guías paso a paso del kernel, consulta la documentación completa:

* 📖 **[Arquitectura e Internals del Kernel](https://github.com/AbdielFritsche/CustomDocker/wiki/Architecture-Internals)**: Detalles sobre `namespaces`, `cgroups v2`, `pivot_root` y capas OverlayFS.
* 📖 **[Referencia Completa del CLI](https://github.com/AbdielFritsche/CustomDocker/wiki/CLI-Reference)**: Tablas de flags y subcomandos (`run`, `start`, `stop`, `exec`, `logs`, `stats`, `ps`, `rm`, `network`).
* 📖 **[Arquitectura de Red y DNS](https://github.com/AbdielFritsche/CustomDocker/wiki/Networking-Deep-Dive)**: Conexión mediante switches `veth`, enrutamiento `iptables` y resolución local.
* 📖 **[Pruebas y Validación Técnica](https://github.com/AbdielFritsche/CustomDocker/wiki/Testing-and-Validation)**: Pruebas de estrés de memoria, fork bombs e inmutabilidad.

---

## Licencia

Distribuido bajo la Licencia MIT. Consulta `LICENSE` para más información.
