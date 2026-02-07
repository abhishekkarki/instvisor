package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/abhishekkarki/instvisor/pkg/collector"
	"github.com/abhishekkarki/instvisor/pkg/config"
	"github.com/abhishekkarki/instvisor/pkg/storage"
)

var (
	configPath = flag.String("config", "/etc/instvisor/agent.yaml", "Path to configuration file")
	version    = "0.1.0"
)

func main() {
	flag.Parse()

	log.Printf("Resource Advisor Agent v%s starting...", version)

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Configuration loaded from: %s", *configPath)

	// Ensure storage directory exists
	if err := os.MkdirAll(filepath.Dir(cfg.Storage.Path), 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// Initialize storage
	store, err := storage.NewSQLiteStorage(cfg.Storage.Path)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	defer func() {
		if err := store.Close(); err != nil {
			log.Printf("Failed to close storage: %v", err)
		}
	}()

	log.Printf("Storage initialized at: %s", cfg.Storage.Path)

	// Initialize collector manager
	manager, err := collector.NewManager(cfg, store)
	if err != nil {
		log.Fatalf("Failed to initialize collector manager: %v", err)
	}

	// Start collectors
	manager.Start()

	// Start cleanup routine
	go runCleanup(store, cfg.Collection.RetentionDays)

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	log.Println("Agent is running. Press Ctrl+C to stop.")
	<-sigChan

	log.Println("Shutdown signal received...")
	manager.Stop()

	log.Println("Agent stopped successfully")
}

func loadConfig(path string) (*config.Config, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Config file not found, using defaults")
		return config.DefaultConfig(), nil
	}

	return config.LoadConfig(path)
}

func runCleanup(store storage.Storage, retentionDays int) {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		log.Printf("Running cleanup: deleting metrics older than %s", cutoff.Format("2006-01-02"))

		if err := store.DeleteOldMetrics(cutoff); err != nil {
			log.Printf("Cleanup error: %v", err)
		} else {
			log.Println("Cleanup completed successfully")
		}
	}
}
