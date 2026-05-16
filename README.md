# Instvisor

<img src="images/instvisor-logo.png" width="95%">

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/abhishekkarki/instvisor)](https://goreportcard.com/report/github.com/abhishekkarki/instvisor)
[![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)](https://golang.org/dl/)
[![GitHub Release](https://img.shields.io/github/v/release/abhishekkarki/instvisor)](https://github.com/abhishekkarki/instvisor/releases)

**Instvisor** is a lightweight Linux host monitoring agent that collects resource metrics over time and generates right-sizing recommendations based on actual usage — helping you reduce cloud costs by 30–70%.

---

## Features

| | |
|---|---|
| **Comprehensive Collection** | CPU, memory, disk I/O, network, and container metrics |
| **Statistical Analysis** | P50 / P90 / P95 / P99 percentiles, standard deviation, workload pattern detection |
| **Instance Recommendations** | Optimal vCPU and RAM sizing for AWS and OTC instance families |
| **Container Visibility** | Per-container CPU and memory breakdown via cgroup v2 — no Docker SDK required |
| **Low Overhead** | < 50 MB RAM, < 1% CPU |
| **Persistent Storage** | SQLite with WAL mode, configurable retention (default: 90 days) |
| **Flexible Deployment** | systemd service, Docker container, or standalone binary |

---

## Installation

### Binary (recommended)

```bash
wget https://github.com/abhishekkarki/instvisor/releases/latest/download/instvisor-linux-amd64.tar.gz
tar -xzf instvisor-linux-amd64.tar.gz
sudo ./install.sh
```

Binaries are available for `linux/amd64` and `linux/arm64`.

### Docker

```bash
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
```

### From Source

Requires Go 1.21+ and `CGO_ENABLED=1` (for SQLite).

```bash
git clone https://github.com/abhishekkarki/instvisor.git
cd instvisor
make build-all
```

---

## Usage

### 1. Start the agent

```bash
# As a systemd service
sudo systemctl enable --now instvisor

# Or run directly
sudo instvisor-agent -config /etc/instvisor/agent.yaml
```

The agent collects metrics every 15 seconds by default and stores them in SQLite.

### 2. Analyze and get recommendations

```bash
# Analyze the last 7 days (default)
sudo instvisor-analyze

# Analyze a longer period
sudo instvisor-analyze -days 30

# Adjust headroom margins
sudo instvisor-analyze -days 7 -headroom-cpu 25 -headroom-mem 20
```

### Example output

```
Current System: prod-server-01 - 8 vCPUs, 32.0 GB RAM (linux/amd64)

=== RESOURCE ANALYSIS ===
Analysis Period: 168h0m0s (7 days)
Workload Pattern: steady_state (confidence: 90%)

CPU Usage:
  Mean:    25.3%
  P95:     42.1%
  Max:     68.0%
  StdDev:  8.4%

Memory Usage:
  Mean:    45.2%
  P95:     52.3%
  Max:     58.0%

=== INSTANCE SIZING RECOMMENDATION ===

Current Configuration:
  vCPUs:   8 cores
  Memory:  32.0 GB

Recommended Configuration:
  vCPUs:   4 cores  (-4 cores)
  Memory:  16.0 GB  (-16.0 GB)

Estimated Resource Savings: 50%

Suggested Instance Types:
  AWS: [m5.xlarge, c5.xlarge]
  OTC: [s3.xlarge.4, c3.xlarge.4]
```

---

## Configuration

Default config is at `/etc/instvisor/agent.yaml`. All fields are optional — sensible defaults apply.

```yaml
collection:
  interval: 15s          # How often metrics are collected
  retention_days: 90     # How long metrics are kept

storage:
  path: /var/lib/instvisor/metrics.db

collectors:
  cpu:
    enabled: true
    per_core: true        # Also collect per-core breakdown
  memory:
    enabled: true
    include_swap: true
  disk:
    enabled: true
    devices: []           # Empty = monitor all block devices
  network:
    enabled: true
    interfaces: []        # Empty = monitor all interfaces (except loopback)
  container:
    enabled: true         # Reads from cgroup v2; requires --privileged in Docker
```

---

## Architecture

```
  ┌────────────────────────────────────────────────────┐
  │                 instvisor-agent                    │
  │                                                    │
  │  ┌─────────┐ ┌─────────┐ ┌──────┐ ┌─────────────┐  │
  │  │  CPU    │ │ Memory  │ │ Disk │ │   Network   │  │
  │  │Collector│ │Collector│ │Coll. │ │  Collector  │  │
  │  └───┬─────┘ └───┬─────┘ └──┬───┘ └──────┬──────┘  │
  │      │           │          │            │         │
  │      └───────────┴────┬─────┘            │         │
  │                       │    ┌─────────────┘         │
  │                       v    v                       │
  │            ┌───────────────────┐   vv              │
  │            │ Collector Manager │  (goroutine/col)  │
  │            └────────┬──────────┘                   │
  │                     │                              │
  │                     v                              │
  │            ┌──────────────────┐                    │
  │            │  SQLite Storage  │  (WAL, 90 days)    │
  │            └──────────────────┘                    │
  └────────────────────────────────────────────────────┘

  ┌────────────────────────────────────────────────────┐
  │                instvisor-analyze                   │
  │                                                    │
  │   SQLite Storage → Analyzer → Recommender          │
  │         (percentiles, pattern detection)           │
  │                     │                              │
  │                     v                              │
  │        Instance sizing report + cloud              │
  │        provider suggestions (AWS / OTC)            │
  └────────────────────────────────────────────────────┘
```

---

## Roadmap

- [x] Core metrics collection (CPU, memory, disk, network)
- [x] Statistical analysis — percentiles, workload pattern detection
- [x] Multi-cloud instance type suggestions (AWS, OTC)
- [x] Container metrics via cgroup v2
- [ ] Prometheus metrics exporter
- [ ] Process-level metrics
- [ ] Web dashboard
- [ ] Kubernetes pod resource recommendations
- [ ] Multi-host fleet analysis
- [ ] Cost estimation with live cloud pricing APIs
- [ ] Alerting system

---

## Contributing

Contributions are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a pull request.

```bash
# Set up
git clone https://github.com/abhishekkarki/instvisor.git
cd instvisor
go mod download

# Build
make build-all

# Test
make test

# Lint
make lint
```

Found a bug or have a question? Open an [issue](https://github.com/abhishekkarki/instvisor/issues) or start a [discussion](https://github.com/abhishekkarki/instvisor/discussions).

---

## License

[MIT](LICENSE) © Abhishek Karki
