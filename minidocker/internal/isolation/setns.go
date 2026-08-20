package isolation

/*
#define _GNU_SOURCE
#include <fcntl.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

__attribute__((constructor)) void init_ns_exec(void) {
    char *target_pid_str = getenv("_VESSEL_EXEC_PID");
    char *cmd_str = getenv("_VESSEL_EXEC_CMD");

    // Si no es una llamada a exec, continuar el flujo normal de Go
    if (!target_pid_str || !cmd_str) {
        return;
    }

    int target_pid = atoi(target_pid_str);
    if (target_pid <= 0) {
        fprintf(stderr, "[vessel-exec] PID inválido: %s\n", target_pid_str);
        exit(1);
    }

    // 1. Unirse a los namespaces en orden
    char *ns_list[] = {"ipc", "uts", "net", "pid", "mnt"};
    char path[128];

    for (int i = 0; i < 5; i++) {
        snprintf(path, sizeof(path), "/proc/%d/ns/%s", target_pid, ns_list[i]);
        int fd = open(path, O_RDONLY);
        if (fd < 0) {
            fprintf(stderr, "[vessel-exec] Error abriendo namespace %s (%s)\n", ns_list[i], path);
            exit(1);
        }
        if (setns(fd, 0) < 0) {
            fprintf(stderr, "[vessel-exec] Error en setns para %s\n", ns_list[i]);
            close(fd);
            exit(1);
        }
        close(fd);
    }

    // 2. Hacer fork para que el hijo nazca dentro del nuevo PID namespace
    pid_t child = fork();
    if (child < 0) {
        perror("[vessel-exec] fork fallo");
        exit(1);
    }

    if (child == 0) {
        // PROCESO HIJO (Dentro del contenedor)
        // Parsear el comando separado por espacios
        char *argv[64];
        int argc = 0;
        char *token = strtok(cmd_str, "\x1f"); // Separador seguro
        while (token != NULL && argc < 63) {
            argv[argc++] = token;
            token = strtok(NULL, "\x1f");
        }
        argv[argc] = NULL;

        char *envp[] = {
            "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
            "TERM=xterm",
            "HOME=/root",
            NULL
        };

        // Ejecutar el binario en el contenedor
        execvpe(argv[0], argv, envp);

        // Fallback: si execvpe no encuentra la ruta absoluta, probar con /bin/sh
        execve(argv[0], argv, envp);
        perror("[vessel-exec] execve fallo");
        exit(127);
    }

    // PROCESO PADRE: Esperar a que el proceso hijo en el contenedor termine
    int status;
    waitpid(child, &status, 0);

    if (WIFEXITED(status)) {
        exit(WEXITSTATUS(status));
    } else if (WIFSIGNALED(status)) {
        exit(128 + WTERMSIG(status));
    }
    exit(0);
}
*/
import "C"
