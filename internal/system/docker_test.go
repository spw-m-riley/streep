package system

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDockerStatusSuccess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script helper is unix-only")
	}

	dir := t.TempDir()
	fakeDocker := filepath.Join(dir, "docker")
	script := "#!/bin/sh\nif [ \"$1\" = \"info\" ]; then\n  exit 0\nfi\nif [ \"$1\" = \"--version\" ]; then\n  echo \"Docker version 26.1.0\"\n  exit 0\nfi\nexit 0\n"
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+":"+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	defer os.Setenv("PATH", oldPath) //nolint:errcheck

	v, err := DockerStatus()
	if err != nil {
		t.Fatalf("DockerStatus() error: %v", err)
	}
	if !strings.Contains(v, "Docker version") {
		t.Fatalf("expected docker version output, got: %q", v)
	}
}

func TestDockerStatusDaemonUnavailable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script helper is unix-only")
	}

	dir := t.TempDir()
	fakeDocker := filepath.Join(dir, "docker")
	script := "#!/bin/sh\nif [ \"$1\" = \"info\" ]; then\n  echo \"daemon unavailable\" 1>&2\n  exit 1\nfi\nif [ \"$1\" = \"--version\" ]; then\n  echo \"Docker version 26.1.0\"\n  exit 0\nfi\nexit 0\n"
	if err := os.WriteFile(fakeDocker, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+":"+oldPath); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	defer os.Setenv("PATH", oldPath) //nolint:errcheck

	_, err := DockerStatus()
	if err == nil {
		t.Fatal("expected daemon unavailable error")
	}
	if !strings.Contains(err.Error(), "docker daemon is not reachable") {
		t.Fatalf("unexpected error: %v", err)
	}
}
