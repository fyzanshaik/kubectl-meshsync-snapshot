package nats

import (
	"fmt"
	"time"

	natsd "github.com/nats-io/nats-server/v2/server"
)
func StartServer() (*natsd.Server, error) {
	fmt.Println("Setting up NATS server options...")
opts := &natsd.Options{
    Host:           "0.0.0.0",     
    Port:           4222,          
    HTTPPort:       8222,          
    ServerName:     "nats",        
    NoLog:          false,
    NoSigs:         true,
    MaxControlLine: 4096,
    Debug:          true,
}
	fmt.Println("Creating NATS server...")
	natsServer, err := natsd.NewServer(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create NATS server: %w", err)
	}
	natsServer.ConfigureLogger()
	fmt.Println("Starting NATS server...")
	go natsServer.Start()
	fmt.Println("Waiting for NATS server to be ready...")
	time.Sleep(100 * time.Millisecond)
	if !natsServer.Running() {
		return nil, fmt.Errorf("NATS server failed to start")
	}
	fmt.Println("NATS server started successfully on port 4222")
	return natsServer, nil
}