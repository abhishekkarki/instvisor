package collector

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/abhishekkarki/instvisor/pkg/metrics"
)

// NetworkCollector collects network metrics
type NetworkCollector struct {
	interval   time.Duration
	interfaces []string
	lastStats  map[string]NetworkStat
}

type NetworkStat struct {
	RxBytes   uint64
	RxPackets uint64
	RxErrors  uint64
	RxDrops   uint64
	TxBytes   uint64
	TxPackets uint64
	TxErrors  uint64
	TxDrops   uint64
}

// NewNetworkCollector creates a new network collector
func NewNetworkCollector(interval time.Duration, interfaces []string) *NetworkCollector {
	return &NetworkCollector{
		interval:   interval,
		interfaces: interfaces,
		lastStats:  make(map[string]NetworkStat),
	}
}

func (n *NetworkCollector) Name() string {
	return "network"
}

func (n *NetworkCollector) Interval() time.Duration {
	return n.interval
}

func (n *NetworkCollector) Collect() ([]metrics.Metric, error) {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/net/dev: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("Failed to close file: %v", err)
		}
	}()

	var result []metrics.Metric
	scanner := bufio.NewScanner(file)
	currentStats := make(map[string]NetworkStat)

	// Skip header lines
	scanner.Scan()
	scanner.Scan()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		iface := strings.TrimSpace(parts[0])

		// Skip loopback
		if iface == "lo" {
			continue
		}

		// Filter by specific interfaces if configured
		if len(n.interfaces) > 0 && !contains(n.interfaces, iface) {
			continue
		}

		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		stat := NetworkStat{
			RxBytes:   parseUint64(fields[0]),
			RxPackets: parseUint64(fields[1]),
			RxErrors:  parseUint64(fields[2]),
			RxDrops:   parseUint64(fields[3]),
			TxBytes:   parseUint64(fields[8]),
			TxPackets: parseUint64(fields[9]),
			TxErrors:  parseUint64(fields[10]),
			TxDrops:   parseUint64(fields[11]),
		}

		currentStats[iface] = stat

		// Calculate rates if we have previous stats
		if lastStat, ok := n.lastStats[iface]; ok {
			netMetrics := n.calculateNetworkMetrics(iface, lastStat, stat)
			result = append(result, netMetrics...)
		}
	}

	// Update last stats
	n.lastStats = currentStats

	return result, scanner.Err()
}

func (n *NetworkCollector) calculateNetworkMetrics(iface string, last, current NetworkStat) []metrics.Metric {
	labels := map[string]string{"interface": iface}
	timeDiff := n.interval.Seconds()

	var result []metrics.Metric

	// Receive bytes/sec
	rxBytesDiff := float64(current.RxBytes - last.RxBytes)
	result = append(result, metrics.NewMetric(
		"network.rx_bytes_per_sec",
		rxBytesDiff/timeDiff,
		"bytes/sec",
		labels,
	))

	// Transmit bytes/sec
	txBytesDiff := float64(current.TxBytes - last.TxBytes)
	result = append(result, metrics.NewMetric(
		"network.tx_bytes_per_sec",
		txBytesDiff/timeDiff,
		"bytes/sec",
		labels,
	))

	// Receive packets/sec
	rxPacketsDiff := float64(current.RxPackets - last.RxPackets)
	result = append(result, metrics.NewMetric(
		"network.rx_packets_per_sec",
		rxPacketsDiff/timeDiff,
		"packets/sec",
		labels,
	))

	// Transmit packets/sec
	txPacketsDiff := float64(current.TxPackets - last.TxPackets)
	result = append(result, metrics.NewMetric(
		"network.tx_packets_per_sec",
		txPacketsDiff/timeDiff,
		"packets/sec",
		labels,
	))

	// Errors
	rxErrorsDiff := float64(current.RxErrors - last.RxErrors)
	result = append(result, metrics.NewMetric(
		"network.rx_errors_per_sec",
		rxErrorsDiff/timeDiff,
		"errors/sec",
		labels,
	))

	txErrorsDiff := float64(current.TxErrors - last.TxErrors)
	result = append(result, metrics.NewMetric(
		"network.tx_errors_per_sec",
		txErrorsDiff/timeDiff,
		"errors/sec",
		labels,
	))

	// Drops
	rxDropsDiff := float64(current.RxDrops - last.RxDrops)
	result = append(result, metrics.NewMetric(
		"network.rx_drops_per_sec",
		rxDropsDiff/timeDiff,
		"drops/sec",
		labels,
	))

	txDropsDiff := float64(current.TxDrops - last.TxDrops)
	result = append(result, metrics.NewMetric(
		"network.tx_drops_per_sec",
		txDropsDiff/timeDiff,
		"drops/sec",
		labels,
	))

	return result
}
