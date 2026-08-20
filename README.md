# Vessel Engine

Vessel es un motor y orquestador de contenedores de bajo nivel implementado en Go que interactúa directamente con el kernel de Linux. Su propósito es proveer aislamiento de procesos, gobernanza estricta de recursos de hardware, redes virtuales aisladas y almacenamiento Copy-on-Write (OverlayFS) sin depender de daemons en segundo plano ni suites externas como Docker o containerd.

El proyecto cuenta con un CLI completo y un motor de composición (`compose up` / `compose down`) capaz de orquestar servicios complejos con bases de datos como MySQL y redirigir tráfico de red bidireccional entre el anfitrión y los contenedores.

---
<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26.6+-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go Version" />
  <img src="https://img.shields.io/badge/Linux_Kernel-Syscalls-FCC624?style=for-the-badge&logo=linux&logoColor=black" alt="Linux Kernel" />
  <img src="https://img.shields.io/badge/cgroups-v2-informational?style=for-the-badge&logo=linux-foundation&logoColor=white" alt="cgroups v2" />
  <img src="https://img.shields.io/badge/OverlayFS-Storage-lightgrey?style=for-the-badge" alt="OverlayFS" />
  <img src="https://img.shields.io/badge/Networking-Bridge%20%7C%20veth%20%7C%20NAT-orange?style=for-the-badge" alt="Networking" />
  <img src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" alt="License MIT" />
</p>

---

> **ADVERTENCIA DE SEGURIDAD / DISCLAIMER** > Este proyecto invoca directamente llamadas al sistema (*syscalls*) de Linux y manipulación de interfaces de red de bajo nivel, por lo que requiere privilegios de superusuario (`sudo`). **No está auditado para entornos de producción.** Se recomienda ejecutarlo en entornos de desarrollo aislados como WSL 2 o máquinas virtuales Linux.
---

## Tabla de Contenidos

1. [Descripción del Proyecto](#descripción-del-proyecto)
2. [Estado del Proyecto](#estado-del-proyecto)
3. [Características Principales](#características-principales)
4. [Estructura del Repositorio](#estructura-del-repositorio)
5. [Requisitos del Sistema](#requisitos-del-sistema)
6. [Instalación y Configuración](#instalación-y-configuración)
7. [Guía de Uso del CLI y Compose](#guía-de-uso-del-cli-y-compose)
8. [Verificación y Casos Reales (MySQL)](#verificación-y-casos-reales-mysql)
9. [Comparativa Técnica](#comparativa-técnica)
10. [Roadmap de Desarrollo](#roadmap-de-desarrollo)
11. [Contribuciones](#contribuciones)
12. [Autor e Inspiración](#autor-e-inspiración)
13. [Licencia](#licencia)

---

## Descripción del Proyecto

Vessel implementa los bloques constructivos de la contenerización moderna utilizando primitivas nativas del kernel de Linux: **Namespaces**, **cgroups v2**, montajes atómicos con **`pivot_root`**, pilas de **OverlayFS** y una pila de red virtual basada en **Linux Bridges**, pares **`veth`** y reglas de enrutamiento con **`iptables`** / proxies TCP en espacio de usuario.

---

## Estado del Proyecto

| Componente | Estado | Implementación Técnica |
| :--- | :--- | :--- |
| **Aislamiento de Procesos** | Completo | Namespaces Linux (`PID`, `UTS`, `Mount`, `IPC`, `NET`) sincronizados vía Unix Pipes |
| **Filesystem Jailing** | Completo | `pivot_root`, bind mounts privados, pseudoterminales `/dev/pts` |
| **Gobernanza de Recursos** | Completo | cgroups v2 (`memory.max`, `pids.max`) integrados en el ciclo de vida |
| **Almacenamiento por Capas** | Completo | Driver OverlayFS (`lowerdir`, `upperdir`, `workdir`, `merged`) desacoplado de la eliminación |
| **Redes Virtuales** | Completo | Linux Bridge (`vessel0`), pares `veth`, asignación de IP estática y proxy de reenvío de puertos |
| **Gestión de Ciclo de Vida** | Completo | CLI con subcomandos `run`, `start`, `stop`, `rm`, `ps` y persistencia de metadata en disco |
| **Orquestación Multi-Servicio** | Completo | Motor Compose (`compose up`, `compose down`) con soporte de rutas personalizadas (`-f`) |
| **Interacción Dinámica** | En Desarrollo | Subcomandos `exec` (`unix.Setns`), `logs` persistentes y `stats` de cgroups v2 |
| **Seguridad Sandbox** | Planificado | Perfiles Seccomp-BPF, Capabilities dropping y User Namespaces (`CLONE_NEWUSER`) |
---

## Características Principales

* **Aislamiento Integral por Namespaces:**
  * `PID`: El proceso corre como PID 1 dentro de su árbol aislado de tareas.
  * `NET`: Pila de red completamente aislada con su propia interfaz `lo` y extremo `veth`.
  * `UTS` & `Mount`: Asignación de hostname (`vessel`) y aislamiento de puntos de montaje mediante propagación `MS_PRIVATE`.
* **Almacenamiento Copy-on-Write (OverlayFS):**
  * Capa base inmutable (`lowerdir`) compartida.
  * Espacio de escritura efímero (`upperdir`) y limpieza atómica coordinada por el gestor de contenedores.
* **Redes Virtuales y Exposición de Puertos:**
  * Creación y gestión de bridges virtuales en el host.
  * Conexión punto a punto mediante pares `veth` inyectados en el namespace de red del subproceso.
  * Exposición de puertos (`-p host:container`) con proxies bidireccionales y soporte de salida a internet mediante `/etc/resolv.conf`.
* **Orquestador Compose Nativo:**
  * Parser y motor de ejecución declarativo para desplegar múltiples servicios interconectados a partir de archivos `compose.yaml` / `docker-compose.yml`.
---

## Estructura del Repositorio

```text
vessel/
├── cmd/
│   └── vessel/                  # Punto de entrada del CLI y subproceso 'init'
│       └── main.go
├── internal/                    # Lógica interna del motor
│   ├── cli/                     # Comandos Cobra (run, start, stop, ps, rm, compose, network)
│   │   ├── compose.go
│   │   ├── exec.go
│   │   ├── network.go
│   │   ├── ps.go
│   │   ├── rm.go
│   │   ├── root.go
│   │   ├── run.go
│   │   ├── start.go
│   │   └── stop.go
│   ├── compose/                 # Motor de parseo y orquestación multi-servicio
│   │   ├── engine.go
│   │   └── spec.go
│   ├── container/               # Gestión de ciclo de vida y metadata
│   │   ├── container.go
│   │   ├── manager.go
│   │   └── process.go
│   ├── isolation/               # Llamadas de bajo nivel al kernel
│   │   ├── cgroups.go           # Controladores cgroups v2
│   │   ├── namespaces.go        # Syscalls clone, unshare y pipes
│   │   └── pivot_root.go        # Montajes y pivot_root
│   ├── network/                 # Pila de redes virtuales
│   │   ├── bridge.go            # Linux bridge
│   │   ├── proxy.go             # Reenvío de puertos TCP
│   │   └── veth.go              # Pares veth y enrutamiento
│   └── storage/                 # Capa de almacenamiento
│       ├── image.go             # Gestión de rootfs e imágenes
│       └── overlayfs.go         # Driver OverlayFS
├── pkg/                         # Paquetes auxiliares reutilizables
│   ├── logger/                  # Logger estructurado
│   └── syscalls/                # Envoltorios de llamadas al sistema
├── go.mod
└── go.sum
```
---

## Requisitos del Sistema

* **Sistema Operativo:** Linux x86_64 nativo o WSL 2 (Ubuntu 22.04+ recomendado).
* **Lenguaje:** Go 1.22 o superior.
* **Subsistema de Control:** cgroups v2 montado en `/sys/fs/cgroup`.
* **Privilegios:** Superusuario (`sudo`) para invocar `syscall.Mount`, `syscall.PivotRoot` y creación de interfaces virtuales.

---

## Instalación y Configuración

**1. Clonar el repositorio:**
```bash
git clone [https://github.com/TuUsuario/vessel.git](https://github.com/TuUsuario/vessel.git)
cd vessel
```

**2. Compilar el binario:**
```bash
go build -o vessel cmd/vessel/main.go
```

**3. Preparar una imagen base (ejemplo con Alpine Linux):**
```bash
mkdir -p assets/alpine
curl -sL [https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz](https://dl-cdn.alpinelinux.org/alpine/v3.19/releases/x86_64/alpine-minirootfs-3.19.1-x86_64.tar.gz) | tar -xz -C assets/alpine
```

---

## Guía de Uso del CLI y Compose

### Comandos de Contenedores

* **Ejecutar un contenedor interactivo con límites de recursos y red:**
  ```bash
  sudo ./vessel run --memory 256m --pids-max 50 -p 8080:80 /bin/sh
  ```

* **Listar contenedores activos y finalizados:**
  ```bash
  sudo ./vessel ps -a
  ```

* **Detener un contenedor en ejecución:**
  ```bash
  sudo ./vessel stop <CONTAINER_ID_O_NOMBRE>
  ```

* **Iniciar un contenedor detenido:**
  ```bash
  sudo ./vessel start <CONTAINER_ID_O_NOMBRE>
  ```

* **Eliminar un contenedor y limpiar sus capas de OverlayFS:**
  ```bash
  sudo ./vessel rm <CONTAINER_ID_O_NOMBRE>
  ```

### Orquestación con Compose

Vessel incluye soporte nativo para levantar stacks completos declarados en YAML:

* **Levantar servicios desde un archivo específico:**
  ```bash
  sudo ./vessel compose up -f deployments/mysql-compose.yaml
  ```

* **Detener y limpiar los servicios y redes creadas:**
  ```bash
  sudo ./vessel compose down -f deployments/mysql-compose.yaml
  ```

---

## Verificación y Casos Reales (MySQL)

Vessel es capaz de aislar y ejecutar cargas de trabajo de bases de datos complejas. Ejemplo de ejecución y validación de **MySQL Community Server**:

1. **Definir el servicio en `compose.yaml`:**
   ```yaml
   version: "3.8"

    networks:
      backend:
        subnet: "192.100.200.0/24"
        gateway: "192.100.200.1"
    
    services:
      database:
        image: "mysql:latest"
        environment:
          - "MYSQL_ALLOW_EMPTY_PASSWORD=yes"
          - "MYSQL_DATABASE=tienda"
          - "MYSQL_USER=admin"
          - "MYSQL_PASSWORD=secret123"
        command:
          - "/bin/sh"
          - "-c"
          - |
            mkdir -p /var/lib/mysql /var/run/mysqld
            chown -R 999:999 /var/lib/mysql /var/run/mysqld
            chmod 777 /var/run/mysqld
            if [ ! -d /var/lib/mysql/mysql ]; then
              mysqld --initialize-insecure --user=mysql --datadir=/var/lib/mysql
              mysqld --user=mysql --datadir=/var/lib/mysql --skip-networking --daemonize
              sleep 3
              mysql -u root -e "CREATE USER IF NOT EXISTS 'root'@'%' IDENTIFIED BY ''; GRANT ALL PRIVILEGES ON *.* TO 'root'@'%' WITH GRANT OPTION; FLUSH PRIVILEGES;"
              mysqladmin -u root shutdown
            fi
            mysqld --user=mysql --datadir=/var/lib/mysql --bind-address=0.0.0.0 --skip-log-bin
        ports:
          - "3306:3306"
        networks:
          backend:
            ipv4_address: "192.100.200.50"
        memory: 1024
        pids_max: 150
   ```

2. **Levantar el servicio:**
   ```bash
   sudo ./vessel compose up -f compose.yaml
   ```

3. **Comprobar conectividad desde el anfitrión:**
   ```bash
   mysql -h 127.0.0.1 -P 3306 -u root -p
   ```

---

### Tabla de Banderas y Parámetros (CLI Flags)

#### 1. Banderas de Creación y Ejecución Directa (`run` / `create`)

| Flag / Parámetro | Shorthand | Tipo | Valor por Defecto | Descripción | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `<comando>` | — | `[]string` | `["/bin/sh"]` | Binario y argumentos a ejecutar dentro del contenedor. | Implementado |
| `--image` | `-i` | `string` | `"assets/alpine"` | Ruta a la imagen base (rootfs) o capa descargada. | Implementado |
| `--port` | `-p` | `string` | `""` | Mapeo de puertos en formato `hostPort:containerPort` (ej. `3306:3306`). | Implementado |
| `--name` | — | `string` | `""` (hash autogenerado) | Nombre único identificador para el contenedor. | Implementado |
| `--memory` | `-m` | `int64` | `0` (sin límite) | Límite máximo de memoria RAM en MB (`memory.max` en cgroups v2). | Implementado |
| `--pids-max` | — | `int64` | `0` (sin límite) | Límite máximo de procesos/hilos simultáneos (`pids.max`). | Implementado |
| `--data-root` | — | `string` | `"/var/lib/minidocker/containers"` | Directorio donde se almacena el estado y capas OverlayFS. | Implementado |
| `--ttl` | — | `duration` | `0` (desactivado) | Tiempo máximo de vida tras el cual el contenedor se autodestruye. | Planificado |

---

#### 2. Subcomandos del Ciclo de Vida, Red y Orquestación

| Subcomando | Parámetro / Flag | Shorthand | Tipo | Valor por Defecto | Descripción | Estado |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **`compose up`** | `--file` | `-f` | `string` | `"minidocker-compose.yml"` | Parsea el YAML, levanta redes bridge y crea/arranca servicios. | Implementado |
| **`compose down`** | `--file` | `-f` | `string` | `"minidocker-compose.yml"` | Detiene servicios asociados al archivo y elimina las redes virtuales. | Implementado |
| **`pull`** | `<imagen:tag>` | — | `string` | *Requerido* | Descarga capas y ensambla un rootfs desde OCI Registry / Docker Hub. | Implementado |
| **`start`** | `<ID>` | — | `string` | *Requerido* | Arranca un contenedor previamente creado o detenido conservando cambios. | Implementado |
| **`stop`** | `<ID_o_Nombre>` | — | `string` | *Requerido* | Envía señal de detención al PID principal del contenedor. | Implementado |
| **`rm`** | `<container_id>` | — | `string` | *Requerido* | Desmonta capas OverlayFS y elimina la metadata en disco. | Implementado |
| **`network rm`** | `<network_name>` | — | `string` | *Requerido* | Desmantela el puente virtual Linux Bridge (`br_<nombre>`). | Implementado |
| **`ps`** | `--data-root` | — | `string` | `"/var/lib/minidocker/containers"` | Lista contenedores registrados, imagen, IPs, puertos y estado. | Implementado |
| **`exec`** | `<ID> <cmd>` | — | `string` | *Requerido* | Inyecta un proceso en los namespaces existentes (`unix.Setns`). | Planificado |
| **`logs`** | `-f`, `--follow` | `-f` | `bool` | `false` | Streaming en vivo de la salida estándar y de errores (`stdout`/`stderr`). | Planificado |

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

## Comparativa Técnica

| Característica | Vessel | Docker | runc / Podman |
| :--- | :--- | :--- | :--- |
| **Arquitectura** | **Daemonless (Ejecución Directa)** | Cliente-Servidor (Daemon `dockerd`) | Daemonless (OCI Runtime) |
| **Aislamiento** | Linux Namespaces + cgroups v2 | libcontainerd + namespaces | libcontainer / cgroups |
| **Filesystem** | OverlayFS nativo en Go | OverlayFS / VFS / Btrfs | Rootfs montado por runtime |
| **Networking** | Linux Bridge + veth + Proxies | Docker Bridge / CNI / IPVS | Netavark / CNI plugins |
| **Dependencias** | **Cero dependencias externas** | Requiere suite de daemons | Requiere herramientas OCI |

---

## Roadmap de Desarrollo

- [x] Namespaces fundamentales (`PID`, `UTS`, `Mount`, `IPC`, `NET`).
- [x] Jailing atómico de filesystem con `pivot_root`.
- [x] Control de recursos con cgroups v2 (`memory.max`, `pids.max`).
- [x] Driver desacoplado de almacenamiento con OverlayFS.
- [x] Red virtual con Linux Bridge, pares `veth` y exposición de puertos.
- [x] CLI avanzada con Cobra (`run`, `start`, `stop`, `rm`, `ps`, `network`).
- [x] Orquestador multi-contenedor (`compose up`, `compose down`).
- [ ] Ejecución de comandos en caliente (`exec`) mediante `unix.Setns`.
- [ ] Streaming y persistencia de registros (`logs -f`).
- [ ] DNS embebido interno para resolución por nombre de servicio en Compose.
- [ ] Cliente nativo de descarga de capas OCI (Docker Hub / Quay.io).
- [ ] Filtros de seguridad Seccomp-BPF y reducción de Linux Capabilities.
- [ ] Soporte Rootless con User Namespaces (`CLONE_NEWUSER`).

---

## Contribuciones

Las contribuciones, reportes de bugs y sugerencias de arquitectura son bienvenidos:

1. Haz un **Fork** del repositorio.
2. Crea tu rama para la nueva feature (///bash git checkout -b feature/nueva-funcionalidad ///).
3. Realiza tus commits detallando las llamadas al sistema involucradas (///bash git commit -m 'feat: soporte para syscall setns en exec' ///).
4. Sube tu rama (///bash git push origin feature/nueva-funcionalidad ///).
5. Abre un **Pull Request** explicando las pruebas realizadas.

---

## Autor e Inspiración

Desarrollado por **Abdiel Fritsche Barajas** como un proyecto de ingeniería de sistemas y kernel de Linux para implementar desde cero las bases que hacen posible la contenerización moderna sin depender de capas de abstracción previas.

---

## Licencia

Este proyecto está bajo la Licencia MIT. Consulta el archivo `LICENSE` para más información.
