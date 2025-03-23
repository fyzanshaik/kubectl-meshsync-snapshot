package nats

import (
	"fmt"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
	natsd "github.com/nats-io/nats-server/v2/server"
)

func StartServer(options *models.Options) (*natsd.Server, error) {
	if !options.QuietMode {
		fmt.Println("Starting temporary NATS server...")
	}
	
	// Configure NATS server options
	opts := &natsd.Options{
		Host:           "0.0.0.0",
		Port:           4222,
		HTTPPort:       8222,            
		ServerName:     "nats",          
		NoLog:          !options.VerboseMode,  // Only log in verbose mode
		NoSigs:         true,
		MaxControlLine: 4096,
		Debug:          options.VerboseMode,
		Trace:          options.VerboseMode,
		Logtime:        true,
		JetStream:      true,
	}
	
	// Create and configure the server
	if options.VerboseMode {
		fmt.Println("Creating NATS server with debug enabled...")
	}
	
	natsServer, err := natsd.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS server: %w", err)
	}
	
	// Only configure logger in verbose mode
	if options.VerboseMode {
		natsServer.ConfigureLogger()
	}
	
	// Start the server in a goroutine
	go natsServer.Start()
	
	// Wait for the server to be ready
	if !options.QuietMode {
		fmt.Print("Waiting for NATS server to be ready...")
	}
	
	if !waitForServerReady(natsServer, 5*time.Second) {
		natsServer.Shutdown()
		return nil, fmt.Errorf("timed out waiting for NATS server to start")
	}
	
	if !options.QuietMode {
		fmt.Println(" ✓")
	}
	
	return natsServer, nil
}

// Wait for the server to be ready with timeout
func waitForServerReady(server *natsd.Server, timeout time.Duration) bool {
	start := time.Now()
	for time.Since(start) < timeout {
		if server.ReadyForConnections(100 * time.Millisecond) {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}