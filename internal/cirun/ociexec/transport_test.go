package ociexec

import (
	"context"
	"strings"
	"testing"
)

func TestDockerTransportRejectsCompletionBeforeExecution(t *testing.T) {
	for _, token := range []string{"__complete", "__completeNoDesc"} {
		t.Run(token, func(t *testing.T) {
			client := &dockerClient{binary: "/nonexistent/docker"}
			err := client.command(context.Background(), "container", "create", token).Run()
			if err == nil || !strings.Contains(err.Error(), "reserved Docker transport") {
				t.Fatalf("transport attempted execution or accepted completion: %v", err)
			}
		})
	}
}
