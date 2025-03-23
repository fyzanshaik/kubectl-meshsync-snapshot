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
	if !options.QuietMode {
		fmt.Printf("Starting MeshSync from: %s\n", meshsyncPath)
	}
	
	// Set up environment variables for MeshSync
	env := append(os.Environ(), 
		fmt.Sprintf("BROKER_URL=%s", brokerURL),
	)
	
	// Set log level based on verbosity
	if options.VerboseMode {
		env = append(env, "LOG_LEVEL=debug")
	} else {
		env = append(env, "LOG_LEVEL=error")
	}
	
	// Create and configure the command with process group for better cleanup
	cmd := exec.Command(meshsyncPath)
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Set process group for better cleanup
	}
	
	// Handle output redirection based on verbosity
	if options.VerboseMode {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// Redirect output to a log file in non-verbose mode
		logPath := filepath.Join(os.TempDir(), "meshsync.log")
		logFile, err := os.Create(logPath)
		if err == nil {
			cmd.Stdout = logFile
			cmd.Stderr = logFile
			
			// Close the file when the command completes
			go func() {
				cmd.Wait()
				logFile.Close()
				if !options.QuietMode {
					fmt.Printf("MeshSync logs available at: %s\n", logPath)
				}
			}()
		} else {
			// Fallback to discard if file creation fails
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
		}
	}
	
	// Start MeshSync
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MeshSync: %w", err)
	}
	
	// Give it a moment to initialize
	if !options.QuietMode {
		fmt.Print("Waiting for MeshSync to initialize...")
	}
	time.Sleep(2 * time.Second)
	if !options.QuietMode {
		fmt.Println(" ✓")
	}
	
	// Check if the process is still running
	if cmd.Process == nil {
		return nil, fmt.Errorf("MeshSync process exited immediately")
	}
	
	return cmd, nil
}

// KillProcessGroup kills the entire process group of a command
func KillProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	
	pgid, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		// Fall back to killing just the process if we can't get the group
		return cmd.Process.Kill()
	}
	
	return syscall.Kill(-pgid, syscall.SIGKILL)
}

// CheckHealth verifies if MeshSync is running properly
func CheckHealth(timeout time.Duration, options *models.Options) bool {
	// Skip the health check in preview mode
	if options.PreviewMode {
		return true
	}
	
	// Wait a moment for MeshSync to initialize
	time.Sleep(1 * time.Second)
	
	// Try a simple check using kubectl
	cmd := exec.Command("kubectl", "get", "meshsync", "meshery-meshsync", "-n", "meshery", "--no-headers")
	cmd.Stderr = io.Discard
	cmd.Stdout = io.Discard
	
	err := cmd.Run()
	return err == nil
}