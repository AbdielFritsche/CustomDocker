cmake_minimum_required(VERSION 3.16)
project(minidocker_project C)

# 1. Habilitar soporte para el compilador de C (necesario para Cgo/GCC)
enable_language(C)

# 2. Localizar el ejecutable de Go en el sistema
find_program(GO_EXECUTABLE go REQUIRED)

# 3. Definir directorio de salida para los binarios (bin/)
set(OUTPUT_BIN_DIR "${CMAKE_BINARY_DIR}/bin")
file(MAKE_DIRECTORY ${OUTPUT_BIN_DIR})

# 4. Configurar variables de compilación de Go
set(GO_ENV ${CMAKE_COMMAND} -E env CGO_ENABLED=1)

# 5. Target: minidockerd (Daemon)
add_custom_command(
    OUTPUT "${OUTPUT_BIN_DIR}/minidockerd"
    COMMAND ${GO_ENV} ${GO_EXECUTABLE} build -o "${OUTPUT_BIN_DIR}/minidockerd" "${CMAKE_CURRENT_SOURCE_DIR}/cmd/minidockerd/main.go"
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    COMMENT "Compilando Daemon minidockerd con CGO_ENABLED=1..."
    VERBATIM
)
add_custom_target(minidockerd ALL DEPENDS "${OUTPUT_BIN_DIR}/minidockerd")

# 6. Target: minidocker (Cliente CLI)
add_custom_command(
    OUTPUT "${OUTPUT_BIN_DIR}/minidocker"
    COMMAND ${GO_ENV} ${GO_EXECUTABLE} build -o "${OUTPUT_BIN_DIR}/minidocker" "${CMAKE_CURRENT_SOURCE_DIR}/cmd/minidocker/main.go"
    WORKING_DIRECTORY ${CMAKE_CURRENT_SOURCE_DIR}
    COMMENT "Compilando Cliente CLI minidocker con CGO_ENABLED=1..."
    VERBATIM
)
add_custom_target(minidocker ALL DEPENDS "${OUTPUT_BIN_DIR}/minidocker")

# 7. Reglas de instalación (Opcional: instala en /usr/local/bin)
install(
    PROGRAMS 
        "${OUTPUT_BIN_DIR}/minidockerd" 
        "${OUTPUT_BIN_DIR}/minidocker"
    DESTINATION /usr/local/bin
)

# Target para arrancar el daemon en background
add_custom_target(start-daemon
    COMMAND sudo nohup "${OUTPUT_BIN_DIR}/minidockerd" > /tmp/minidockerd.log 2>&1 &
    DEPENDS minidockerd
    COMMENT "Iniciando minidockerd en segundo plano (logs en /tmp/minidockerd.log)..."
)

# Target para detener el daemon
add_custom_target(stop-daemon
    COMMAND sudo pkill -SIGTERM minidockerd || true
    COMMAND sudo rm -f /var/run/minidocker.sock
    COMMENT "Deteniendo minidockerd y limpiando socket..."
)

# Target de prueba rápida completa (compila, inicia daemon si no corre y ejecuta ps)
add_custom_target(test-ps
    COMMAND sudo "${OUTPUT_BIN_DIR}/minidocker" ps
    DEPENDS minidocker
)