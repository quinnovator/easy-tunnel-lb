package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/quinnovator/easy-tunnel-lb/internal/api_client"
	"github.com/quinnovator/easy-tunnel-lb/internal/config"
	"github.com/quinnovator/easy-tunnel-lb/internal/controller"
	"github.com/quinnovator/easy-tunnel-lb/internal/k8s"
	"github.com/quinnovator/easy-tunnel-lb/internal/tunnel"
	"github.com/quinnovator/easy-tunnel-lb/internal/utils"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	// Initialize logger
	logger := utils.NewLogger(cfg.LogLevel)

	// Create Kubernetes client
	k8sClient, err := k8s.NewClient()
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Failed to create Kubernetes client")
		os.Exit(1)
	}

	// Create API client
	apiClient := api_client.NewClient(cfg.ServerURL, cfg.APIKey)

	// Create tunnel manager
	tunnelMgr := tunnel.NewManager()

	// Create reconciler
	reconciler := controller.NewServiceReconciler(k8sClient, apiClient, tunnelMgr, logger)

	// Create service watcher
	watcher := controller.NewServiceWatcher(k8sClient, reconciler, logger)

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		logger.WithFields(map[string]interface{}{
			"signal": sig.String(),
		}).Info("Received shutdown signal")
		cancel()
	}()

	// Start the watcher
	logger.Info("Starting easy-tunnel-lb controller")
	if err := watcher.Start(ctx); err != nil {
		logger.WithFields(map[string]interface{}{
			"error": err.Error(),
		}).Error("Controller failed")
		os.Exit(1)
	}
} 