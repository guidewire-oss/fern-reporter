package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/guidewire/fern-reporter/pkg/utils"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure cleanup on exit

	// Start gRPC server in a goroutine
	go StartGRPCServer(ctx)

	// Listen for OS signals (CTRL+C, SIGTERM)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	// Use select to wait for either a signal or a timeout
	<-sigs // Wait for a shutdown signal (CTRL+C)
	utils.Log.Info("[gRPC-LOG]: Received shutdown signal, stopping gRPC server...")
	cancel() // Cancel the context to gracefully stop the server

}
