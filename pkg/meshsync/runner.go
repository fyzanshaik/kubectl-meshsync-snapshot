package meshsync

import (
	"fmt"
	"os"
	"os/exec"
)
func Run(brokerURL, meshsyncPath string) (*exec.Cmd, error) {
	cmd := exec.Command(meshsyncPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("BROKER_URL=%s", brokerURL))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MeshSync: %w", err)
	}
	return cmd, nil
}