package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/crds"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/meshsync"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/nats"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/snapshot"
	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/utils"
)

func main() {
	// Parse command line flags
	options := models.NewDefaultOptions()
	
	flag.StringVar(&options.OutputFile, "output", options.OutputFile, "Output file for the snapshot")
	flag.StringVar(&options.OutputFile, "o", options.OutputFile, "Output file for the snapshot (shorthand)")
	flag.BoolVar(&options.AutoName, "auto-name", options.AutoName, "Generate filename with timestamp")
	flag.StringVar(&options.Namespace, "namespace", options.Namespace, "Filter resources by namespace")
	flag.StringVar(&options.Namespace, "n", options.Namespace, "Filter resources by namespace (shorthand)")
	flag.StringVar(&options.ResourceType, "type", options.ResourceType, "Filter resources by type (e.g., pods, deployments)")
	flag.StringVar(&options.ResourceType, "t", options.ResourceType, "Filter resources by type (shorthand)")
	flag.StringVar(&options.LabelSelector, "selector", options.LabelSelector, "Filter resources by label selector (e.g., app=nginx)")
	flag.StringVar(&options.LabelSelector, "l", options.LabelSelector, "Filter resources by label selector (shorthand)")
	flag.StringVar(&options.OutputFormat, "format", options.OutputFormat, "Output format: json or yaml (default \"json\")")
	flag.BoolVar(&options.FastMode, "fast", options.FastMode, "Capture only essential resources")
	
	waitTime := flag.Int("time", int(options.CollectionTime.Seconds()), "Collection time in seconds")
	flag.BoolVar(&options.QuietMode, "quiet", options.QuietMode, "Minimal output")
	flag.BoolVar(&options.QuietMode, "q", options.QuietMode, "Minimal output (shorthand)")
	flag.BoolVar(&options.VerboseMode, "verbose", options.VerboseMode, "Detailed output")
	flag.BoolVar(&options.VerboseMode, "v", options.VerboseMode, "Detailed output (shorthand)")
	flag.BoolVar(&options.PreviewMode, "preview", options.PreviewMode, "Show what would be captured without saving")
	
	excludeStr := flag.String("exclude", "", "Comma-separated list of resource types to exclude")
	
	flag.Parse()
	
	// Update collection time from flag
	options.CollectionTime = time.Duration(*waitTime) * time.Second
	
	// Process exclude list
	if *excludeStr != "" {
		options.ExcludeTypes = strings.Split(*excludeStr, ",")
		for i, t := range options.ExcludeTypes {
			options.ExcludeTypes[i] = strings.TrimSpace(t)
		}
	}
	
	// Process auto-name
	if options.AutoName {
		options.OutputFile = utils.GenerateTimestampedFilename(options.OutputFile)
	}
	
	// Create the context with timeout for cleaner process management
	ctx, cancel := context.WithTimeout(context.Background(), options.CollectionTime+time.Duration(5)*time.Second)
	defer cancel()
	
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	go func() {
		<-sigChan
		fmt.Println("\nInterrupted. Cleaning up...")
		cancel() // Cancel context to trigger cleanup
	}()
	
	if !options.QuietMode {
		fmt.Println("Starting kubectl meshsync-snapshot...")
	}
	
	// Skip resource discovery in preview mode
	if options.PreviewMode {
		fmt.Println("Preview mode - showing what would be captured without actually running")
		previewResources, _ := meshsync.CollectResources(ctx, "", options)
		utils.PrintResourceSummary(previewResources, options)
		fmt.Println("Preview completed. No snapshot was created.")
		return
	}
	
	// Find MeshSync binary
	meshsyncPath, err := findMeshSyncBinary()
	if err != nil {
		fmt.Printf("Error finding MeshSync binary: %v\n", err)
		os.Exit(1)
	}
	
	// Set up host entries for NATS
	if err := setupHostsEntry(); err != nil && !options.QuietMode {
		fmt.Printf("Warning: Failed to set up hosts entry, MeshSync may fail: %v\n", err)
	}
	
	// Start NATS server
	natsServer, err := nats.StartServer(options)
	if err != nil {
		fmt.Printf("Error starting NATS server: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if natsServer != nil {
			if !options.QuietMode {
				fmt.Println("Shutting down NATS server...")
			}
			natsServer.Shutdown()
		}
	}()
	
	// Apply CRDs and create instances
	crdManager := crds.NewManager("pkg/crds/meshery-crds.yaml", options)
	if err := crdManager.Apply(); err != nil {
		fmt.Printf("Error applying CRDs: %v\n", err)
		os.Exit(1)
	}
	defer crdManager.Remove()
	
	// Start MeshSync
	meshSyncCmd, err := meshsync.Run("nats:4222", meshsyncPath, options)
	if err != nil {
		fmt.Printf("Error starting MeshSync: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if meshSyncCmd != nil && meshSyncCmd.Process != nil {
			if !options.QuietMode {
				fmt.Println("Terminating MeshSync process...")
			}
			meshsync.KillProcessGroup(meshSyncCmd)
		}
	}()
	
	// Check if MeshSync is healthy
	if !meshsync.CheckHealth(10*time.Second, options) && !options.QuietMode {
		fmt.Println("Warning: MeshSync may not be fully initialized, but will try to continue...")
	}
	
	// Collect resources
	resources, err := meshsync.CollectResources(ctx, "nats://localhost:4222", options)
	if err != nil {
		fmt.Printf("Error collecting resources: %v\n", err)
		os.Exit(1)
	}
	
	// Get absolute path for output file
	absOutputPath, err := filepath.Abs(options.OutputFile)
	if err != nil {
		absOutputPath = options.OutputFile
	}
	
	// Create parent directories if they don't exist
	if err := utils.CreateParentDirs(absOutputPath); err != nil {
		fmt.Printf("Warning: Could not create parent directories: %v\n", err)
	}
	
	// Save snapshot to file
	if !options.QuietMode {
		fmt.Printf("Saving snapshot to %s...\n", absOutputPath)
	}
	
	if err := snapshot.SaveToFile(resources, absOutputPath, options); err != nil {
		fmt.Printf("Error saving snapshot: %v\n", err)
		os.Exit(1)
	}
	
	// Try to read the file to confirm it was written correctly
	if _, err := os.Stat(absOutputPath); err != nil {
		fmt.Printf("Warning: Could not confirm file was created: %v\n", err)
	} else {
		fileInfo, err := os.Stat(absOutputPath)
		if err == nil && !options.QuietMode {
			fmt.Printf("Snapshot file size: %s\n", utils.FormatSize(fileInfo.Size()))
		}
	}
	
	// Print resource summary
	utils.PrintResourceSummary(resources, options)
	
	if !options.QuietMode {
		fmt.Printf("Snapshot created successfully with %d resources\n", len(resources))
		fmt.Printf("You can now import this snapshot into Meshery\n")
		fmt.Printf("Snapshot saved to: %s\n", absOutputPath)
	}
}

func setupHostsEntry() error {
	// Check if entry already exists
	checkCmd := exec.Command("grep", "-q", "nats", "/etc/hosts")
	if checkCmd.Run() == nil {
		// Entry already exists
		return nil
	}
	
	// This is a hacky approach for development only
	cmd := exec.Command("sudo", "sh", "-c", "echo '127.0.0.1 nats' >> /etc/hosts")
	return cmd.Run()
}

func findMeshSyncBinary() (string, error) {
	// Check in current directory
	if _, err := os.Stat("./meshsync"); err == nil {
		return "./meshsync", nil
	}
	
	// Check relative to executable
	execPath, err := os.Executable()
	if err == nil {
		meshsyncPath := filepath.Join(filepath.Dir(execPath), "meshsync")
		if _, err := os.Stat(meshsyncPath); err == nil {
			return meshsyncPath, nil
		}
	}
	
	// Check in PATH
	meshsyncPath, err := exec.LookPath("meshsync")
	if err == nil {
		return meshsyncPath, nil
	}
	
	return "", fmt.Errorf("MeshSync binary not found. Please ensure it's in the same directory as this plugin or in your PATH")
}