# MiniDocker (Alpha)

MiniDocker es un motor de contenedores de bajo nivel implementado en Go que interactúa directamente con el kernel de Linux. Su propósito es proveer aislamiento de procesos, gobernanza estricta de recursos de hardware y almacenamiento Copy-on-Write sin depender de daemons externos ni suites como Docker o containerd.

El proyecto está diseñado como una plataforma para micro-sandboxing, evaluación de código de baja latencia y observabilidad nativa del kernel.

---
<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" />
  <img src="https://img.shields.io/badge/Linux_Kernel-Syscalls-FCC624?style=for-the-badge&logo=linux&logoColor=black" alt="Linux Kernel" />
  <img src="https://img.shields.io/badge/cgroups-v2-informational?style=for-the-badge&logo=linux-foundation&logoColor=white" alt="cgroups v2" />
  <img src="https://img.shields.io/badge/OverlayFS-Storage-lightgrey?style=for-the-badge" alt="OverlayFS" />
  <img src="https://img.shields.io/badge/version-v0.2.0--dev-blue?style=for-the-badge" alt="Version Alpha" />
  <img src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" alt="License MIT" />
</p>

---

> **ADVERTENCIA DE SEGURIDAD / DISCLAIMER**  
> Este proyecto se encuentra en fase **experimental**. MiniDocker invoca directamente llamadas al sistema (*syscalls*) privilegiadas y requiere ejecutarse con permisos de superusuario (`sudo`). **No está diseñado ni auditado para su uso en entornos de producción.** Se recomienda encarecidamente ejecutarlo dentro de una máquina virtual (VM) o un entorno aislado como WSL 2 para mitigar cualquier riesgo de configuración errónea sobre el sistema anfitrión.

---

## Tabla de Contenidos

1. [Descripción del Proyecto](#descripción-del-proyecto)
2. [Estado del Proyecto](#estado-del-proyecto)
3. [Características Principales](#características-principales)
4. [Estructura del Repositorio](#estructura-del-repositorio)
5. [Requisitos del Sistema](#requisitos-del-sistema)
6. [Instalación y Configuración](#instalación-y-configuración)
7. [Guía de Uso y Opciones del CLI](#guía-de-uso-y-opciones-del-cli)
8. [Verificación y Pruebas](#verificación-y-pruebas)
9. [Diferenciadores y Casos de Uso](#diferenciadores-y-casos-de-uso)
10. [Roadmap de Desarrollo](#roadmap-de-desarrollo)
11. [Contribuciones](#contribuciones)
12. [Autor e Inspiración](#autor-e-inspiración)
13. [Licencia](#licencia)

---

## Descripción del Proyecto

MiniDocker implementa los bloques constructivos fundamentales de la contenerización moderna utilizando únicamente llamadas al sistema (*syscalls*) de Linux y la jerarquía unificada de **cgroups v2**. Permite ejecutar entornos aislados, reproducibles y seguros con tiempos de arranque mínimos.

---

## Estado del Proyecto

**Fase Actual:** `Alpha v0.2.0-dev` (En desarrollo activo)

| Componente | Estado | Implementación Técnica |
| :--- | :--- | :--- |
| Aislamiento de Procesos | Completo | Linux Namespaces (`PID`, `UTS`, `Mount`, `IPC`) |
| Filesystem Jailing | Completo | `pivot_root`, bind mounts privados, pseudoterminales `/dev/pts` |
| Gobernanza de Recursos | Completo | cgroups v2 (`memory.max`, `pids.max`) con controladores de subárbol |
| Capas de Almacenamiento | Completo | Driver OverlayFS (`lowerdir`, `upperdir`, `workdir`, `merged`) |
| Persistencia y Metadata | En Desarrollo | Registro del ciclo de vida en JSON (`/var/lib/minidocker/state.json`) |
| Redes Virtuales | En Desarrollo | Linux Bridge (`minibr0`) + pares de interfaces `veth` + NAT con `iptables` |
| Seguridad Sandbox | Planificado | Perfiles Seccomp-BPF y Namespaces de Usuario (`CLONE_NEWUSER`) |
| Telemetría y CLI | Planificado | Subcomando `stats`, exportador Prometheus y TUI |

---

## Características Principales

* **Aislamiento por Linux Namespaces:**
  * `PID`: El proceso principal del contenedor se ejecuta como PID 1 en su propio espacio de nombres.
  * `UTS`: Asignación de hostname independiente sin afectar al sistema anfitrión.
  * `Mount (NS)`: Espacio privado de puntos de montaje mediante propagación `MS_PRIVATE`.
  * `IPC`: Segmentación de memoria compartida, colas de mensajes y semáforos POSIX/SysV.
* **Confinamiento Seguro del Sistema de Archivos (`pivot_root`):**
  * Reemplazo atómico de la raíz del sistema de archivos mediante `syscall.PivotRoot`.
  * Montaje independiente de `/proc` y pseudoterminales `/dev/pts` para soporte interactivo de terminales.
* **Control de Recursos con cgroups v2:**
  * Prevención de ataques de denegación de servicio (*fork bombs*) mediante la restricción de subprocesos con `pids.max`.
  * Control del límite de memoria con `memory.max` y activación automática del *OOM Killer* del kernel.
* **Almacenamiento por Capas (OverlayFS):**
  * Preservación e inmutabilidad del sistema de archivos base de solo lectura (`lowerdir`).
  * Espacio de escritura efímero (`upperdir`) y limpieza atómica al finalizar la ejecución del contenedor.

---

## Estructura del Repositorio

```text
minidocker/
├── cmd/
│   ├── minidocker/          # Punto de entrada del CLI principal
│   │   └── main.go
│   └── runtimed/            # (Opcional) Daemon de ejecución interno
│       └── main.go
├── internal/                # Lógica privada del runtime
│   ├── cli/                 # Definición de comandos y parsing (Cobra)
│   │   ├── run.go
│   │   ├── exec.go
│   │   └── root.go
│   ├── container/           # Dominio y orquestación del contenedor
│   │   ├── container.go     # Estructuras principales e interfaces
│   │   ├── process.go       # Manejo del ciclo de vida del proceso
│   │   └── manager.go       # State & container store (índices/metadata)
│   ├── isolation/           # Interacción de bajo nivel con el Kernel
│   │   ├── namespaces.go    # Flags y syscalls (PID, UTS, MNT, NET, IPC)
│   │   ├── cgroups.go       # Manipulación de /sys/fs/cgroup (cgroups v2)
│   │   └── pivot_root.go    # Syscalls de cambio de rootfs
│   ├── storage/             # Gestión de capas de sistema de archivos
│   │   ├── overlayfs.go     # Montajes OverlayFS (lowerdir, upperdir, merged)
│   │   └── image.go         # Descarga y extracción de rootfs
│   └── network/             # Configuración de red
│       ├── bridge.go        # Linux bridge (br0)
│       └── veth.go          # Interfaces veth y NAT
├── pkg/                     # Paquetes reutilizables o utilidades públicas
│   ├── logger/              # Logger personalizado con middleware/decoradores
│   └── syscalls/            # Wrappers de bajo nivel
├── go.mod
└── go.sum
```
---

## Comparativa Técnica

| Herramienta | Nivel / Propósito | Dependencias Externas | Enfoque Principal |
| :--- | :--- | :--- | :--- |
| **MiniDocker** | Runtime ligero / Educativo | Ninguna (Go puro + Linux Kernel) | Aprendizaje de kernel internals, sandboxing efímero y observabilidad nativa. |
| **Docker** | Plataforma completa (PaaS) | containerd, runc, dockerd | Ecosistema integral de desarrollo, empaquetado y despliegue de microservicios. |
| **containerd** | Daemon de gestión de ciclo de vida | runc, OCI specifications | Gestión intermedia de imágenes, almacenamiento y supervisión de procesos. |
| **runc** | Runtime OCI de bajo nivel | Linux kernel, libcontainer | Especificación estándar OCI para crear y correr contenedores a partir de bundles. |
| **gVisor** | Sandbox con Kernel en User-Space | Sentry, Gofer | Aislamiento extremo mediante intercepción de syscalls en espacio de usuario. |

---

## Requisitos del sistema

* **Sistema Operativo:** Linux de 64 bits o WSL 2 (Ubuntu 20.04+ recomendado).
* **Lenguaje:** Go 1.20 o superior.
* **Subsistema** del Kernel: cgroups v2 montado en /sys/fs/cgroup.
* **Permisos:** Privilegios de superusuario (sudo) para ejecutar llamadas al sistema privilegiadas.
---

## Instalación y Configuración

**Clonar el repositorio localmente:**

```Bash
git clone https://github.com/AbdielFritsche/CustomDocker
cd CustomDocker/minidocker

```
**Descargar y descomprimir la imagen base de prueba (Alpine Linux):**
```Bash
mkdir -p assets/alpine
curl -sL https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz | tar -xz -C assets/alpine

```

**Compilar el binario del runtime:**

```Bash
go build -o minidocker cmd/minidocker/main.go
```
---

## Guía de Uso y Opciones del CLI

### Sintaxis General

```bash
sudo ./minidocker run [FLAGS] <COMANDO> [ARGUMENTOS...]
```

### Tabla de Banderas (CLI Flags)

| Flag / Parámetro | Tipo | Valor por Defecto | Descripción | Estado |
| :--- | :--- | :--- | :--- | :--- |
| `<comando>` | `string` | `/bin/sh` | Binario a ejecutar dentro del contenedor aislado. | Implementado |
| `--memory`, `-m` | `string` | `100m` | Límite máximo de RAM (`memory.max` en cgroups v2). | Planificado (Cobra CLI) |
| `--pids-max` | `int` | `20` | Número máximo de tareas y procesos permitidos (`pids.max`). | Planificado (Cobra CLI) |
| `--name` | `string` | `aleatorio` | Nombre identificador para el contenedor y su cgroup. | Planificado (Cobra CLI) |
| `--ttl` | `duration` | `0` (desactivado) | Tiempo máximo de vida tras el cual el contenedor se destruye. | Planificado |

### Ejemplos de Ejecución

* **Sesión interactiva en shell:**
  ```bash
  sudo ./minidocker run /bin/sh
  ```

* **Ejecución directa de un comando:**
  ```bash
  sudo ./minidocker run /bin/ls -la /
  ```

---

## Verificación y Pruebas

### 1. Pruebas Automatizadas (Unit / Integration Tests)
Actualmente, el proyecto no cuenta con una suite formal de pruebas unitarias debido a la dependencia directa de llamadas al sistema del kernel de Linux y privilegios administrativos.
```bash
# Ejecución del runner de pruebas de Go (en desarrollo para v0.3.0)
go test -v ./...
```

### 2. Pruebas Manuales de Validación Técnica

* **Aislamiento de Procesos (PID Namespace):**  
  Verifique que la sesión actual se ejecute bajo el PID 1:
  ```sh
  ps aux
  ```

* **Aislamiento de Hostname (UTS Namespace):**  
  Compruebe el nombre de host asignado dentro del espacio de nombres:
  ```sh
  hostname
  # Salida esperada: minidocker
  ```

* **Prueba de Restricción de Procesos (cgroups v2):**  
  Intente superar el límite configurado de procesos concurrentes para detonar el bloqueo:
  ```sh
  for i in $(seq 1 20); do sleep 60 & done
  # Salida esperada: sh: can't fork: Resource temporarily unavailable
  ```

* **Prueba de Inmutabilidad del Filesystem (OverlayFS):**  
  Genere un archivo dentro del contenedor y verifique la persistencia tras salir:
  ```sh
  # Dentro del contenedor:
  echo "prueba de aislamiento" > /test.txt
  exit
  
  # En el host:
  ls assets/alpine/test.txt
  # Salida esperada: No such file or directory (la imagen base no fue modificada)
  ```

---

## Diferenciadores y Casos de Uso
* **Sandboxing para Evaluación de Código (Runners):** Optimizado para entornos efímeros (tipo plataformas de evaluación técnica o funciones serverless) con tiempo de inicio cercano a cero y autodestrucción por tiempo de vida (--ttl).
* **Seguridad de Confianza Cero (Zero-Trust):** Arquitectura pensada para integrar perfiles de Seccomp-BPF y reducción de capacidades administrativas del kernel (CAP_SYS_ADMIN) por defecto.
* **Observabilidad de Bajo Nivel:** Lectura directa de métricas de rendimiento desde cgroups v2 (memory.current, cpu.stat, io.stat) sin overhead de daemons secundarios.
---

## Roadmap de Desarrollo

- [x] Implementación de Namespaces base (`PID`, `UTS`, `Mount`, `IPC`).
- [x] Confinamiento de raíz seguro mediante `pivot_root`.
- [x] Control de recursos con cgroups v2 (`pids.max`, `memory.max`).
- [x] Sistema de archivos por capas con OverlayFS.
- [ ] Migración completa a Cobra CLI con soporte formal para flags (`--memory`, `--pids-max`, `--ttl`).
- [ ] Interfaz de línea de comandos avanzada con subcomandos `ps`, `stop` y `rm`.
- [ ] Conectividad de red mediante Linux Bridge y pares `veth`.
- [ ] Modo Rootless con Namespaces de Usuario (`CLONE_NEWUSER`).
- [ ] Integración de perfiles Seccomp para filtrado de syscalls.
- [ ] Suite de pruebas de integración automatizadas.
- [ ] Cliente nativo de OCI Registry para descarga directa de imágenes.

---

## Contribuciones

Las contribuciones, reportes de bugs y sugerencias de arquitectura son bienvenidos. Si deseas contribuir:

1. Haz un **Fork** del repositorio.
2. Crea una rama para tu feature o fix (```bash git checkout -b feature/nueva-funcionalidad```).
3. Realiza tus commits con mensajes descriptivos (```bash git commit -m 'feat: agregar soporte para limite de CPU'```).
4. Haz push a tu rama (```bash git push origin feature/nueva-funcionalidad```).
5. Abre un **Pull Request** detallando los cambios y las pruebas técnicas realizadas.

---

## Autor e Inspiración

Desarrollado por **Abdiel Fritsche Barajas** como un proyecto de profundización e ingeniería de sistemas para comprender a fondo el funcionamiento interno del kernel de Linux, la implementación de llamadas al sistema y los mecanismos que hacen posible la contenerización moderna sin depender de capas de abstracción preexistentes.

---

## Licencia

Este proyecto está distribuido bajo la licencia MIT. Consulte el archivo `LICENSE` para más detalles.
