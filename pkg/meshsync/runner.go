package meshsync

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
)
func Run(brokerURL, meshsyncPath string, options *models.Options) (*exec.Cmd, error) {
	if options.VerboseMode {
		fmt.Printf("Starting MeshSync from: %s\n", meshsyncPath)
	}
	if _, err := os.Stat(meshsyncPath); err != nil {
		return nil, fmt.Errorf("MeshSync binary not found at %s: %w", meshsyncPath, err)
	}
	env := append(os.Environ(), 
		fmt.Sprintf("BROKER_URL=%s", brokerURL),
		"LOG_LEVEL=fatal", 
		"MESHKIT_LOG_LEVEL=fatal", 
	)
	cmd := exec.Command(meshsyncPath)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, 
	}
	if options.VerboseMode {
		logPath := filepath.Join(os.TempDir(), "meshsync.log")
		logFile, err := os.Create(logPath)
		if err == nil {
			cmd.Stdout = logFile
			cmd.Stderr = logFile
			go func() {
				cmd.Wait()
				logFile.Close()
				fmt.Printf("MeshSync logs available at: %s\n", logPath)
			}()
		} else {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
	} else {
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MeshSync: %w", err)
	}
	if options.VerboseMode {
		fmt.Print("Waiting for MeshSync to initialize...")
	}
	time.Sleep(2 * time.Second)
	if options.VerboseMode {
		fmt.Println(" ✓")
	}
	if cmd.Process == nil {
		return nil, fmt.Errorf("MeshSync process exited immediately")
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		return nil, fmt.Errorf("MeshSync process is not running: %w", err)
	}
	return cmd, nil
}
func KillProcessGroup(cmd *exec.Cmd) error {
    if cmd == nil || cmd.Process == nil {
        return nil
    }
    pid := cmd.Process.Pid
    pgid, err := syscall.Getpgid(pid)
    if err == nil {
        if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
            cmd.Process.Kill()
        }
    } else {
        cmd.Process.Kill()
    }
    exec.Command("pkill", "-9", "-f", "meshsync").Run() 
    time.Sleep(500 * time.Millisecond)
    return nil
}
func CheckHealth(timeout time.Duration, options *models.Options) bool {
    startTime := time.Now()
    for time.Since(startTime) < timeout {
        cmd := exec.Command("kubectl", "get", "meshsync", "meshery-meshsync", "-n", "meshery", "--no-headers")
        cmd.Stderr = io.Discard
        cmd.Stdout = io.Discard
        if err := cmd.Run(); err == nil {
            return true
        }
        time.Sleep(300 * time.Millisecond)
    }
    return false
}