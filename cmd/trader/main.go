package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/signalalpha/weex-ai-trading/internal/config"
	"github.com/signalalpha/weex-ai-trading/internal/monitor"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := monitor.NewLogger(cfg.Log.Level, cfg.Log.Output)
	logger.Info("Starting WEEX AI Trading system...")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	// TODO: Initialize and start trading system
	// - Initialize API client
	// - Initialize data collector
	// - Initialize strategy engine
	// - Initialize execution engine
	// - Initialize risk manager
	// - Start trading loop

	logger.Info("Trading system initialized. Waiting for shutdown signal...")

	// Wait for context cancellation
	<-ctx.Done()
	logger.Info("Trading system stopped.")
}
