// main.go
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"server-monitor/config"
	"server-monitor/monitor"

	"go.uber.org/zap"
)

func main() {
	// Parse command line arguments
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Set up logging
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("can't initialize zap logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig(logger.Named("config"), *configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and start the monitoring service
	monitorService := monitor.NewMonitorService(logger.Named("monitor"), cfg)
	if err := monitorService.Start(); err != nil {
		logger.Error("Failed to start monitoring service", zap.Error(err))
		panic(err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-sigChan
	logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))

	// Stop the monitoring service
	monitorService.Stop()
	logger.Info("Monitoring service stopped")
}
