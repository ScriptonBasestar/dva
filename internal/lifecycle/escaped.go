package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
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

// HaltedFile marks that `dva stop` terminated the entry's process itself (TASK-632).
// haltProcess keeps the pidfile for a quick restart, so without this marker a dead pid plus
// an answering health port is ambiguous: after a deliberate stop, whatever answers now (the
// same server run by hand, or any other program) is not a server that escaped dva.
func (p *PluginContext) HaltedFile(name string) string {
	return filepath.Join(p.StateDir(config.PidsDirName), name+".halted")
}

// markHalted records a deliberate stop; a failure only loses the false-positive guard, so it
// is logged, not returned — the stop itself succeeded.
func markHalted(pctx *PluginContext, name string) {
	if err := os.WriteFile(pctx.HaltedFile(name), nil, 0o644); err != nil && pctx.Logger != nil {
		pctx.Logger.Debug("record halted marker", "name", name, "error", err)
	}
}

// wasHalted reports whether dva stopped the entry's tracked process itself.
func wasHalted(pctx *PluginContext, name string) bool {
	_, err := os.Stat(pctx.HaltedFile(name))
	return err == nil
}

// clearHalted forgets a deliberate stop once a new process is tracked or state is removed.
func clearHalted(pctx *PluginContext, name string) {
	_ = os.Remove(pctx.HaltedFile(name))
}
