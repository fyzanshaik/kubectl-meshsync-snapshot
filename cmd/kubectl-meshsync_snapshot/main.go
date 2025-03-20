package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/crds"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/meshsync"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/nats"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/snapshot"
)

func main() {
	outputFile := flag.String("output", "meshsync-snapshot.json", "Output file for the snapshot")
	waitTime := flag.Int("wait", 30, "Time to wait for resource collection (seconds)")
	flag.Parse()

	fmt.Println("Starting kubectl meshsync-snapshot...")

	meshsyncPath, err := findMeshSyncBinary()
	if err != nil {
		fmt.Printf("Error finding MeshSync binary: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting temporary NATS server...")
	natsServer, err := nats.StartServer()
	if err != nil {
		fmt.Printf("Error starting NATS server: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if natsServer != nil {
			fmt.Println("Shutting down NATS server...")
			natsServer.Shutdown()
		}
	}()

	if err := setupHostsEntry(); err != nil {
		fmt.Printf("Warning: Failed to set up hosts entry, MeshSync may fail: %v\n", err)
	}
	crdManager := crds.NewManager("pkg/crds/meshery-crds.yaml")
if err := crdManager.Apply(); err != nil {
    fmt.Printf("Error applying CRDs: %v\n", err)
    os.Exit(1)
}
defer crdManager.Remove()
	fmt.Println("Starting MeshSync...")
	meshSyncCmd, err := meshsync.Run("nats:4222", meshsyncPath)
	if err != nil {
		fmt.Printf("Error starting MeshSync: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if meshSyncCmd != nil && meshSyncCmd.Process != nil {
			meshSyncCmd.Process.Kill()
		}
	}()

	fmt.Printf("Collecting resources for %d seconds...\n", *waitTime)
	resources, err := meshsync.CollectResources("nats://localhost:4222", time.Duration(*waitTime)*time.Second)
	if err != nil {
		fmt.Printf("Error collecting resources: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Saving snapshot to %s...\n", *outputFile)
	if err := snapshot.SaveToFile(resources, *outputFile); err != nil {
		fmt.Printf("Error saving snapshot: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Snapshot created successfully with %d resources\n", len(resources))
}

func setupHostsEntry() error {
    // This is a hacky approach for development only
    cmd := exec.Command("sudo", "sh", "-c", "echo '127.0.0.1 nats' >> /etc/hosts")
    return cmd.Run()
}

func findMeshSyncBinary() (string, error) {
	if _, err := os.Stat("./meshsync"); err == nil {
		return "./meshsync", nil
	}

	execPath, err := os.Executable()
	if err == nil {
		meshsyncPath := filepath.Join(filepath.Dir(execPath), "meshsync")
		if _, err := os.Stat(meshsyncPath); err == nil {
			return meshsyncPath, nil
		}
	}

	meshsyncPath, err := exec.LookPath("meshsync")
	if err == nil {
		return meshsyncPath, nil
	}

	return "", fmt.Errorf("MeshSync binary not found. Please ensure it's in the same directory as this plugin or in your PATH")
}
