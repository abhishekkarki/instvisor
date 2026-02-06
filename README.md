# Instvisor

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/abhishekkarki/instvisor)](https://goreportcard.com/report/github.com/abhishekkarki/instvisor)

**Instvisor** is a lightweight system resource monitoring agent that analyzes machine performance over time and provides optimal instance sizing recommendations based on actual usage patterns.

Similar to cAdvisor for containers, Instvisor monitors your entire host and helps you right-size your infrastructure, potentially saving 30-70% on cloud costs.

## Features

- 🔍 **Comprehensive Metrics Collection**: CPU, Memory, Disk I/O, Network
- 📊 **Statistical Analysis**: P50/P90/P95/P99 percentiles, workload pattern detection
- 💰 **Cost Optimization**: Instance sizing recommendations for AWS, OTC, Azure
- 🎯 **Low Overhead**: <50MB RAM, <1% CPU usage
- 💾 **SQLite Storage**: Efficient time-series data with configurable retention
- 🐳 **Container Support**: Run as systemd service or Docker container
- 📈 **Prometheus Ready**: Export metrics to existing monitoring stack (coming soon)

## Quick Start

### Binary Installation
```bash
# Download latest release
wget https://github.com/abhishekkarki/instvisor/releases/latest/download/instvisor-linux-amd64.tar.gz

# Extract
tar -xzf instvisor-linux-amd64.tar.gz

# Install
sudo ./install.sh
```

### Docker
```bash
# Run agent
docker run -d \
  --name instvisor \
  --privileged \
  --pid=host \
  -v /:/rootfs:ro \
  -v /var/run:/var/run:ro \
  -v /sys:/sys:ro \
  -v /var/lib/docker:/var/lib/docker:ro \
  -v instvisor-data:/var/lib/instvisor \
  abhishekkarki/instvisor:latest

# Analyze collected data
docker exec instvisor instvisor-analyze
```

### From Source
```bash
# Clone repository
git clone https://github.com/abhishekkarki/instvisor.git
cd instvisor

# Build
make build

# Run
sudo ./build/instvisor-agent
```

## Usage

### Start the Agent
```bash
# As systemd service
sudo systemctl start instvisor
sudo systemctl enable instvisor

# Or run directly
sudo instvisor-agent -config /etc/instvisor/agent.yaml
```

### Analyze and Get Recommendations
```bash
# Analyze last 7 days
sudo instvisor-analyze

# Analyze specific period
sudo instvisor-analyze -days 30

# With custom headroom
sudo instvisor-analyze -days 7 -headroom-cpu 25 -headroom-mem 20
```

### Example Output
```
Current System: prod-server-01 - 8 vCPUs, 32.0 GB RAM (linux/amd64)

=== RESOURCE ANALYSIS ===
Analysis Period: 168h0m0s (7 days)
Workload Pattern: steady_state (confidence: 90%)

CPU Usage:
  Mean:   25.3%
  P95:    42.1%
  Max:    68.0%

Memory Usage:
  Mean:   45.2%
  P95:    52.3%
  Max:    58.0%

=== INSTANCE SIZING RECOMMENDATION ===

Current Configuration:
  vCPUs:  8 cores
  Memory: 32.0 GB

Recommended Configuration:
  vCPUs:  4 cores (-4 cores)
  Memory: 16.0 GB (-16.0 GB)

💰 Estimated Resource Savings: 50%

Suggested Instance Types:
  AWS: [m5.xlarge, c5.xlarge]
  OTC: [s3.xlarge.4, c3.xlarge.4]
```

## Configuration

Edit `/etc/instvisor/agent.yaml`:
```yaml
collection:
  interval: 15s          # Collection frequency
  retention_days: 90     # Data retention period

storage:
  path: /var/lib/instvisor/metrics.db

collectors:
  cpu:
    enabled: true
    per_core: true
  memory:
    enabled: true
    include_swap: true
  disk:
    enabled: true
    devices: []          # Empty = all devices
  network:
    enabled: true
    interfaces: []       # Empty = all interfaces
```

## Architecture
```
┌─────────────────────────────────────┐
│   Collectors (CPU, Mem, Disk, Net)  │
│              ↓                      │
│        Collector Manager            │
│              ↓                      │
│       SQLite Storage                │
│              ↓                      │
│        Analysis Engine              │
│              ↓                      │
│    Recommendations + Reports        │
└─────────────────────────────────────┘
```

## Contributing

We love contributions! Please read our [Contributing Guide](CONTRIBUTING.md) to get started.

### Development Setup
```bash
# Clone repository
git clone https://github.com/abhishekkarki/instvisor.git
cd instvisor

# Install dependencies
go mod download

# Run tests
make test

# Build
make build

# Run locally
make dev
```

### Running Tests
```bash
# Unit tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Roadmap

- [x] Core metrics collection (CPU, Memory, Disk, Network)
- [x] Statistical analysis and recommendations
- [x] Multi-cloud instance type suggestions
- [ ] Prometheus metrics exporter
- [ ] Container/cgroup v2 support
- [ ] Process-level metrics
- [ ] Web dashboard
- [ ] Kubernetes pod resource recommendations
- [ ] Multi-host fleet analysis
- [ ] Cost estimation with cloud pricing APIs
- [ ] Alerting system

## FAQ

**Q: How is this different from cAdvisor?**  
A: cAdvisor monitors containers, Instvisor monitors entire hosts and provides sizing recommendations. They complement each other.

**Q: Does it work on ARM?**  
A: Yes! We provide binaries for amd64 and arm64.

**Q: How much data does it store?**  
A: With default settings (15s interval, 90 days retention), approximately 2-5GB per host.

**Q: Can I export metrics to Prometheus?**  
A: Prometheus exporter is planned for v0.2.0.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Support

- 📖 [Documentation](https://github.com/abhishekkarki/instvisor/wiki)
- 🐛 [Issue Tracker](https://github.com/abhishekkarki/instvisor/issues)
- 💬 [Discussions](https://github.com/abhishekkarki/instvisor/discussions)

## Acknowledgments

Inspired by:
- [cAdvisor](https://github.com/google/cadvisor)
- [node_exporter](https://github.com/prometheus/node_exporter)
- [Kubernetes Vertical Pod Autoscaler](https://github.com/kubernetes/autoscaler)