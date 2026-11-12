package lifecycle

import (
	"log/slog"
	"slices"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// TestComposePsIncludesStopped binds TASK-605: a compose project whose containers all exited or
// crashed must come back from the status query as services that are not running, not as an
// empty list. docker compose ps needs -a for that; podman-compose ps always passes -a to
// `podman ps` and rejects -a itself, so its query must not add one.
func TestComposePsIncludesStopped(t *testing.T) {
	env := config.NewEnvironment(nil, "/tmp", "/tmp")

	t.Run("docker compose ps lists stopped containers", func(t *testing.T) {
		pctx := &PluginContext{
			Entry:  &config.LifecycleEntry{Name: "db", Compose: &config.ComposePluginConfig{Files: []string{"compose.yml"}}},
			Env:    env,
			Logger: slog.Default(),
		}
		_, args, err := (&ComposePlugin{}).statusQueryArgs(pctx)
		if err != nil {
			t.Fatal(err)
		}
		ps := slices.Index(args, "ps")
		if ps < 0 || !slices.Contains(args[ps:], "-a") {
			t.Fatalf("docker compose status query does not list stopped containers: %v", args)
		}
		if !slices.Equal(args[ps:], []string{"ps", "-a", "--format", "json"}) {
			t.Fatalf("docker compose status query args = %v, want ps -a --format json", args[ps:])
		}
	})

	t.Run("podman-compose ps uses the same rule without a flag it rejects", func(t *testing.T) {
		pctx := &PluginContext{
			Entry:  &config.LifecycleEntry{Name: "db", PodmanCompose: &config.PodmanComposePluginConfig{Files: []string{"compose.yml"}}},
			Env:    env,
			Logger: slog.Default(),
		}
		_, args := (&PodmanComposePlugin{}).statusQueryArgs(pctx)
		ps := slices.Index(args, "ps")
		if ps < 0 || !slices.Equal(args[ps:], composePsArgs(podmanComposePsAllFlag)) {
			t.Fatalf("podman-compose status query args = %v, want the shared ps args", args)
		}
		if slices.Contains(args[ps:], "-a") {
			t.Fatalf("podman-compose ps has no -a option and would fail on it: %v", args)
		}
	})

	t.Run("an exited project parses to not-running services", func(t *testing.T) {
		out := []byte(`[{"Service":"db","State":"exited","ExitCode":0},{"Service":"api","State":"exited","ExitCode":137}]`)
		services, err := parseComposeServicesJSON(out)
		if err != nil {
			t.Fatal(err)
		}
		if len(services) != 2 {
			t.Fatalf("exited project yielded %d services, want 2: %+v", len(services), services)
		}
		for _, s := range services {
			if s.State != "exited" || serviceLooksRunning(s.State) {
				t.Errorf("service %q state %q, want a not-running exited service", s.Name, s.State)
			}
		}
	})

	t.Run("a completed one-shot beside a running service is left out, a crashed one is kept", func(t *testing.T) {
		out := []byte(`{"Service":"web","State":"running","Health":"healthy"}
{"Service":"migrate","State":"exited","ExitCode":0}
{"Service":"worker","State":"exited","ExitCode":1}`)
		services, err := parseComposeServicesJSON(out)
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, s := range services {
			names = append(names, s.Name)
		}
		if !slices.Equal(names, []string{"web", "worker"}) {
			t.Fatalf("services = %v, want [web worker] (completed migrate dropped, crashed worker kept)", names)
		}
	})
}

// TestParseComposeServicesJSONServiceName binds the podman-compose status path: `podman ps
// --format json` has no Service field, so the service is only in Labels
// (io.podman.compose.service) or, failing that, the container name. docker compose ps reports
// Service directly and writes Labels as a "k=v,k=v" string, so both encodings must parse
// without a JSON error. The one-shot filter (exit 0 beside a running container) ignores the
// name and must keep working on podman-shaped records.
func TestParseComposeServicesJSONServiceName(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want []string
	}{
		{
			name: "docker compose array keeps Service",
			out:  `[{"Service":"db","State":"running"},{"Service":"api","State":"running"}]`,
			want: []string{"db", "api"},
		},
		{
			name: "podman array reads the compose service label, not the container name",
			out:  `[{"Names":["proj_db_1"],"Labels":{"io.podman.compose.service":"db","com.docker.compose.service":"db","com.docker.compose.project":"proj"},"State":"running"}]`,
			want: []string{"db"},
		},
		{
			name: "podman array without a compose label falls back to Names[0]",
			out:  `[{"Names":["proj_db_1"],"State":"running","ExitCode":0}]`,
			want: []string{"proj_db_1"},
		},
		{
			name: "docker comma-separated Labels string resolves when Service is absent",
			out:  `[{"Names":["proj-api-1"],"Labels":"com.docker.compose.project=proj,com.docker.compose.service=api","State":"running"}]`,
			want: []string{"api"},
		},
		{
			name: "podman JSON lines resolve names and still drop a completed one-shot",
			out: `{"Names":["proj_db_1"],"Labels":{"io.podman.compose.service":"db"},"State":"running"}
{"Names":["proj_migrate_1"],"Labels":{"io.podman.compose.service":"migrate"},"State":"exited","ExitCode":0}
{"Names":["proj_worker_1"],"Labels":{"io.podman.compose.service":"worker"},"State":"exited","ExitCode":1}`,
			want: []string{"db", "worker"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			services, err := parseComposeServicesJSON([]byte(tc.out))
			if err != nil {
				t.Fatalf("parseComposeServicesJSON: %v", err)
			}
			var names []string
			for _, s := range services {
				names = append(names, s.Name)
			}
			if !slices.Equal(names, tc.want) {
				t.Fatalf("resolved service names = %v, want %v", names, tc.want)
			}
		})
	}
}
