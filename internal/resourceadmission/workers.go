package resourceadmission

import (
	"strconv"
	"strings"
)

// CapWorkerEnvironment bounds known numeric worker variables cooperatively.
// Unknown syntax and tools that ignore these variables remain outside this limit.
func CapWorkerEnvironment(env []string, limit int) []string {
	if limit < 1 {
		return env
	}
	keys := map[string]bool{"GOMAXPROCS": true, "CARGO_BUILD_JOBS": true, "RUST_TEST_THREADS": true, "PYTEST_MAX_WORKERS": true}
	values := map[string]string{}
	order := []string{}
	for _, v := range env {
		k, x, ok := strings.Cut(v, "=")
		if !ok {
			continue
		}
		if _, seen := values[k]; !seen {
			order = append(order, k)
		}
		values[k] = x
	}
	for key := range keys {
		value, seen := values[key]
		if !seen || value == "" {
			values[key] = strconv.Itoa(limit)
			if !seen {
				order = append(order, key)
			}
			continue
		}
		if n, err := strconv.Atoi(value); err == nil && n > limit {
			values[key] = strconv.Itoa(limit)
		}
	}
	out := make([]string, 0, len(order))
	for _, key := range order {
		out = append(out, key+"="+values[key])
	}
	return out
}
