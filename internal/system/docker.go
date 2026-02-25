package system

import (
	"fmt"
	"os/exec"
	"strings"
)

// DockerStatus checks whether Docker is installed and the daemon is reachable.
// It returns the docker version string on success.
func DockerStatus() (string, error) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return "", fmt.Errorf("docker not found in PATH")
	}

	infoCmd := exec.Command(dockerPath, "info")
	if err := infoCmd.Run(); err != nil {
		return "", fmt.Errorf("docker daemon is not reachable: %w", err)
	}

	versionCmd := exec.Command(dockerPath, "--version")
	out, err := versionCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read docker version: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}
