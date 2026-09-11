package runner

import (
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestInteractionTreeDestructiveInheritance(t *testing.T) {
	tTrue := true
	tFalse := false

	entries := map[string]*config.InteractionCommand{
		"db": {
			Description: "Database tasks",
			Destructive: &tTrue,
			Subcommands: map[string]*config.InteractionCommand{
				"reset": {
					Description: "Reset DB (inherits destructive)",
					Command:     "rake db:reset",
				},
				"migrate": {
					Description: "Migrate DB (explicitly non-destructive override)",
					Command:     "rake db:migrate",
					Destructive: &tFalse,
				},
			},
		},
		"cache": {
			Description: "Cache tasks",
			Subcommands: map[string]*config.InteractionCommand{
				"status": {
					Description: "Cache status",
					Command:     "redis-cli ping",
				},
				"flush": {
					Description: "Flush cache (explicitly destructive)",
					Command:     "redis-cli flushall",
					Destructive: &tTrue,
				},
			},
		},
	}

	tree := NewInteractionTree(entries)

	// Top-level db: destructive
	dbCmd := tree.Find("db")
	if dbCmd == nil || !dbCmd.Destructive {
		t.Errorf("db.Destructive = %v, want true", dbCmd != nil && dbCmd.Destructive)
	}

	// Subcommand db reset: inherits destructive = true
	dbResetCmd := tree.Find("db", "reset")
	if dbResetCmd == nil || !dbResetCmd.Destructive {
		t.Errorf("db reset Destructive = %v, want true (inherited)", dbResetCmd != nil && dbResetCmd.Destructive)
	}

	// Subcommand db migrate: child explicitly false
	dbMigrateCmd := tree.Find("db", "migrate")
	if dbMigrateCmd == nil || dbMigrateCmd.Destructive {
		t.Errorf("db migrate Destructive = %v, want false (overridden)", dbMigrateCmd != nil && dbMigrateCmd.Destructive)
	}

	// Subcommand cache status: false
	cacheStatusCmd := tree.Find("cache", "status")
	if cacheStatusCmd == nil || cacheStatusCmd.Destructive {
		t.Errorf("cache status Destructive = %v, want false", cacheStatusCmd != nil && cacheStatusCmd.Destructive)
	}

	// Subcommand cache flush: child explicitly true
	cacheFlushCmd := tree.Find("cache", "flush")
	if cacheFlushCmd == nil || !cacheFlushCmd.Destructive {
		t.Errorf("cache flush Destructive = %v, want true", cacheFlushCmd != nil && cacheFlushCmd.Destructive)
	}
}
