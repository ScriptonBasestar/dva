/* Fixed-size kernel fixture. Compiled and executed only inside verified OCI. */
#include <errno.h>
#include <signal.h>
#include <stdio.h>
#include <string.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

int main(int argc, char **argv) {
    if (argc == 2 && strcmp(argv[1], "detach") == 0) {
        int ready[2];
        if (pipe(ready) != 0) return 2;
        pid_t child = fork();
        if (child < 0) return 3;
        if (child == 0) {
            close(ready[0]);
            if (setsid() < 0) _exit(4);
            pid_t orphan = fork();
            if (orphan < 0) _exit(5);
            if (orphan > 0) _exit(0);
            if (write(ready[1], "R", 1) != 1) _exit(6);
            close(ready[1]);
            for (;;) sleep(30);
        }
        close(ready[1]);
        char marker = 0;
        if (read(ready[0], &marker, 1) != 1 || marker != 'R') return 7;
        close(ready[0]);
        if (waitpid(child, NULL, 0) != child) return 8;
        puts("DVA_OCI_CHILD_READY");
        fflush(stdout);
        for (;;) sleep(30);
    }
    pid_t children[24];
    int count = 0, denied = 0;
    for (int i = 0; i < 24; i++) {
        pid_t child = fork();
        if (child < 0) { denied++; continue; }
        if (child == 0) { sleep(30); _exit(0); }
        children[count++] = child;
    }
    unsigned long max_events = 0, value = 0;
    char key[32];
    FILE *events = fopen("/sys/fs/cgroup/pids.events", "r");
    if (events != NULL) {
        while (fscanf(events, "%31s %lu", key, &value) == 2) {
            if (strcmp(key, "max") == 0) max_events = value;
        }
        fclose(events);
    }
    for (int i = 0; i < count; i++) kill(children[i], SIGTERM);
    while (waitpid(-1, NULL, 0) > 0 || errno == EINTR) errno = 0;
    printf("DVA_OCI_PID_PROBE denied=%d max=%lu\n", denied, max_events);
    return denied > 0 && max_events > 0 ? 0 : 9;
}
