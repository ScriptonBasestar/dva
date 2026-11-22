package lifecycle

import (
	"fmt"
	"strings"
)

// escapedExtraKey is the ServiceStatus.Extra key holding the addresses an escaped server
// still answers on.
const escapedExtraKey = "escaped"

// escapedError reports a process entry whose tracked pid exited while its health addresses
// (heldHealthEndpoints) still answer: the server outlived the process dva started — it
// daemonized (Astro's dev server backgrounds itself by default) or detached from the process
// group — so dva no longer tracks what serves the entry.
//
// dva never kills it: the only identity left is the port, and a port is not proof of
// ownership (TASK-629, user decision B1).
func escapedError(name string, pid int, addresses []string) error {
	return fmt.Errorf("entry %q escaped from tracking: its process (pid %d) exited but %s still answers — the server daemonized or detached, so dva cannot stop it; stop it by PID and run it in the foreground (e.g. ASTRO_DEV_BACKGROUND=0)",
		name, pid, strings.Join(addresses, ", "))
}
