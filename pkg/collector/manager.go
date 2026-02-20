package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/abhishekkarki/instvisor/pkg/config"
	"github.com/abhishekkarki/instvisor/pkg/storage"
)

// Manager manages all collectors and coordinates metric collection
type Manager struct {
	collectors []Collector
	storage    storage.Storage
	interval   time.Duration
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewManager creates a new collector manager
func NewManager(cfg *config.Config, store storage.Storage) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		collectors: make([]Collector, 0),
		storage:    store,
		interval:   cfg.Collection.Interval,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize collectors based on configuration
	if cfg.Collectors.CPU.Enabled {
		manager.collectors = append(manager.collectors,
			NewCPUCollector(cfg.Collection.Interval, cfg.Collectors.CPU.PerCore))
	}

	if cfg.Collectors.Memory.Enabled {
		manager.collectors = append(manager.collectors,
			NewMemoryCollector(cfg.Collection.Interval, cfg.Collectors.Memory.IncludeSwap))
	}

	if cfg.Collectors.Disk.Enabled {
		manager.collectors = append(manager.collectors,
			NewDiskCollector(cfg.Collection.Interval, cfg.Collectors.Disk.Devices))
	}

	if cfg.Collectors.Network.Enabled {
		manager.collectors = append(manager.collectors,
			NewNetworkCollector(cfg.Collection.Interval, cfg.Collectors.Network.Interfaces))
	}

	if cfg.Collectors.Container.Enabled {
		containerCollector := NewContainerCollector(cfg.Collection.Interval)
		manager.collectors = append(manager.collectors, containerCollector)
	}

	if len(manager.collectors) == 0 {
		return nil, fmt.Errorf("no collectors enabled")
	}

	return manager, nil
}

// Start begins metric collection
func (m *Manager) Start() {
	log.Printf("Starting collector manager with %d collectors", len(m.collectors))

	for _, collector := range m.collectors {
		m.wg.Add(1)
		go m.runCollector(collector)
	}
}

// Stop stops all collectors
func (m *Manager) Stop() {
	log.Println("Stopping collector manager...")
	m.cancel()
	m.wg.Wait()
	log.Println("All collectors stopped")
}

func (m *Manager) runCollector(collector Collector) {
	defer m.wg.Done()

	ticker := time.NewTicker(collector.Interval())
	defer ticker.Stop()

	log.Printf("Starting collector: %s (interval: %s)", collector.Name(), collector.Interval())

	// Collect immediately on start
	m.collectAndStore(collector)

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.collectAndStore(collector)
		}
	}
}

func (m *Manager) collectAndStore(collector Collector) {
	metricsData, err := collector.Collect()
	if err != nil {
		log.Printf("Error collecting metrics from %s: %v", collector.Name(), err)
		return
	}

	if len(metricsData) == 0 {
		return
	}

	if err := m.storage.WriteMetrics(metricsData); err != nil {
		log.Printf("Error storing metrics from %s: %v", collector.Name(), err)
		return
	}

	log.Printf("Collected and stored %d metrics from %s", len(metricsData), collector.Name())
}
