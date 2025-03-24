package nats

import (
	"fmt"
	"net"
	"time"

	"github.com/fyzanshaik/kubectl-meshsync_snapshot/pkg/models"
	natsd "github.com/nats-io/nats-server/v2/server"
)

func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func StartServer(options *models.Options) (*natsd.Server, error) {
	if options.VerboseMode {
		fmt.Println("Starting temporary NATS server...")
	}

	if isPortInUse(4222) {
		return nil, fmt.Errorf("port 4222 is already in use, cannot start NATS server")
	}

	opts := &natsd.Options{
		Host:           "0.0.0.0",
		Port:           4222,
		HTTPPort:       8222,            
		ServerName:     "nats",          
		NoLog:          true,  
		NoSigs:         true,
		MaxControlLine: 4096,
		Debug:          false,
		Trace:          false,
		Logtime:        false,
		JetStream:      false,
	}

	natsServer, err := natsd.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS server: %w", err)
	}

	if !options.VerboseMode {
		natsServer.SetLogger(nil, false, false)
	}

	go natsServer.Start()

	if options.VerboseMode {
		fmt.Print("Waiting for NATS server to be ready...")
	}

	if !waitForServerReady(natsServer, 5*time.Second) {
		natsServer.Shutdown()
		return nil, fmt.Errorf("timed out waiting for NATS server to start")
	}

	if options.VerboseMode {
		fmt.Println(" ✓")
	}

	return natsServer, nil
}

func waitForServerReady(server *natsd.Server, timeout time.Duration) bool {
	start := time.Now()
	for time.Since(start) < timeout {
		if server.ReadyForConnections(50 * time.Millisecond) {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}