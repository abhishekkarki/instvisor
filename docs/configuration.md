# Configuration Reference

This document provides a complete reference for all instvisor configuration options.

---

## Configuration File Location

Instvisor looks for its configuration file in the following locations (in order):

1. Path specified via `-config` flag: `instvisor-agent -config /path/to/config.yaml`
2. Default location: `/etc/instvisor/agent.yaml`
3. Current directory: `./agent.yaml`

**Recommended:** Use the default location `/etc/instvisor/agent.yaml` for system-wide installations.

---

## Configuration File Format

Instvisor uses YAML format. Here's the complete configuration with all available options:
```yaml
# Collection settings
collection:
  # How often to collect metrics
  # Valid values: duration string (e.g., "5s", "15s", "30s", "1m")
  # Default: 15s
  # Recommendation: 15s for production, 5s for debugging, 30s for low-resource systems
  interval: 15s

  # How long to keep metrics in the database
  # Valid values: integer (days)
  # Default: 90
  # Note: 90 days with 15s interval ≈ 5GB disk space
  retention_days: 90

# Storage settings
storage:
  # Path to SQLite database file
  # Default: /var/lib/instvisor/metrics.db
  # Note: Ensure this directory exists and is writable by instvisor user
  path: /var/lib/instvisor/metrics.db

# Collector settings
collectors:

  # CPU metrics collector
  cpu:
    # Enable/disable CPU metrics collection
    # Default: true
    enabled: true

    # Collect per-core metrics in addition to aggregate
    # Default: true
    # Set to false on systems with many cores (64+) to reduce storage
    per_core: true

  # Memory metrics collector
  memory:
    # Enable/disable memory metrics collection
    # Default: true
    enabled: true

    # Include swap metrics (swap usage, swap in/out rates)
    # Default: true
    # Set to false if swap is disabled on your system
    include_swap: true

  # Disk I/O metrics collector
  disk:
    # Enable/disable disk metrics collection
    # Default: true
    enabled: true

    # List of specific devices to monitor (empty = all devices)
    # Default: [] (monitor all)
    # Examples: ["sda", "nvme0n1"]
    # Use: df -h to see device names
    devices: []

  # Network metrics collector
  network:
    # Enable/disable network metrics collection
    # Default: true
    enabled: true

    # List of specific interfaces to monitor (empty = all interfaces)
    # Default: [] (monitor all)
    # Examples: ["eth0", "ens3"]
    # Use: ip addr to see interface names
    interfaces: []

  # Process metrics collector (future feature)
  process:
    # Enable/disable per-process metrics
    # Default: false
    # Note: Currently not implemented
    enabled: false

    # Number of top processes to track
    # Default: 10
    top_n: 10

  # Container metrics collector
  container:
    # Enable/disable container metrics collection
    # Default: true
    # Note: Requires Docker and cgroup v2
    enabled: true

# Server settings (future feature - Prometheus exporter)
server:
  # Enable/disable HTTP metrics endpoint
  # Default: false
  # Note: Currently not implemented
  enabled: false

  # Port for metrics endpoint
  # Default: 9090
  port: 9090
```

---

## Configuration Options Explained

### Collection Settings

#### `collection.interval`

**Type:** Duration string
**Default:** `15s`
**Valid values:** `5s`, `10s`, `15s`, `30s`, `1m`, `5m`

How frequently instvisor collects metrics from the system.

**Impact on storage:**
| Interval | Metrics/day | 90-day storage |
|----------|-------------|----------------|
| 5s       | 17,280      | ~15 GB         |
| 15s      | 5,760       | ~5 GB          |
| 30s      | 2,880       | ~2.5 GB        |
| 1m       | 1,440       | ~1.2 GB        |

**When to adjust:**
- **Shorter (5s)**: High-frequency workloads, debugging, need detailed view
- **Standard (15s)**: Production systems, good balance
- **Longer (30s-1m)**: Low-resource systems, long-term trends only

**Trade-offs:**
- Shorter = more granular data, catch brief spikes
- Shorter = more disk space, higher CPU/IO overhead
- Longer = less overhead, smaller database
- Longer = may miss brief spikes

---

#### `collection.retention_days`

**Type:** Integer
**Default:** `90`
**Valid values:** `1` to `365`

How many days of historical metrics to keep in the database. Older metrics are automatically deleted.

**Recommended values:**
- **7 days**: Minimal, good for weekly right-sizing checks
- **30 days**: Standard, captures monthly patterns
- **90 days**: Recommended, captures quarterly trends
- **180+ days**: Long-term analysis, seasonal patterns

**Disk space calculation:**
```
Size ≈ (131 metrics × bytes_per_row × samples_per_day × retention_days)
     ≈ (131 × 200 bytes × 5760 × retention_days)
     ≈ (150 MB × retention_days)

Examples:
- 7 days   ≈ 1 GB
- 30 days  ≈ 4.5 GB
- 90 days  ≈ 13.5 GB
```

**Note:** Database is compacted nightly. Actual size may be smaller due to SQLite compression.

---

### Storage Settings

#### `storage.path`

**Type:** String (file path)
**Default:** `/var/lib/instvisor/metrics.db`

Location of the SQLite database file.

**Requirements:**
- Directory must exist
- Directory must be writable by instvisor user
- Sufficient disk space (see retention calculation above)
- Fast disk recommended (SSD preferred for frequent writes)

**Examples:**
```yaml
# Standard installation
storage:
  path: /var/lib/instvisor/metrics.db

# Custom location
storage:
  path: /data/monitoring/instvisor.db

# Docker volume
storage:
  path: /var/lib/instvisor/metrics.db  # mapped to named volume
```

---

### Collector Settings

#### `collectors.cpu.enabled`

**Type:** Boolean
**Default:** `true`

Enable or disable CPU metrics collection.

**Collected metrics when enabled:**
- `cpu.usage` - Per-CPU and aggregate usage percentage
- `cpu.user` - Time spent in user space
- `cpu.system` - Time spent in kernel space
- `cpu.idle` - Idle time
- `cpu.iowait` - Time waiting for I/O

**Disable if:** You only care about memory/disk, or CPU is always low.

---

#### `collectors.cpu.per_core`

**Type:** Boolean
**Default:** `true`

Collect metrics for each CPU core individually in addition to aggregate.

**When enabled:**
- Metrics for: `cpu`, `cpu0`, `cpu1`, `cpu2`, ... (total + per core)
- Storage: ~4x more data on 4-core system
- Use case: Identify single-threaded bottlenecks, uneven load distribution

**When to disable:**
- Systems with many cores (64+)
- Only care about total CPU usage
- Storage space is limited

**Example scenario:**
```
With per_core=true:
  cpu (total): 50%
  cpu0: 90%  ← Single-threaded bottleneck!
  cpu1: 40%
  cpu2: 40%
  cpu3: 30%

Insight: Application needs multithreading, not more cores
```

---

#### `collectors.memory.enabled`

**Type:** Boolean
**Default:** `true`

Enable or disable memory metrics collection.

**Collected metrics when enabled:**
- `memory.total` - Total system memory
- `memory.available` - Available for new processes
- `memory.used` - Currently used
- `memory.usage_percent` - Percentage used
- `memory.buffers` - Buffers cache
- `memory.cached` - Page cache

**Disable if:** Memory is never constrained, or you only care about CPU.

---

#### `collectors.memory.include_swap`

**Type:** Boolean
**Default:** `true`

Include swap space metrics in collection.

**Additional metrics when enabled:**
- `memory.swap_total` - Total swap space
- `memory.swap_used` - Used swap space
- `memory.swap_free` - Free swap space

**When to disable:**
- Swap is disabled on your system (`swapon -s` shows nothing)
- You don't use swap in production (best practice for databases)
- Container environments where swap is disabled

---

#### `collectors.disk.enabled`

**Type:** Boolean
**Default:** `true`

Enable or disable disk I/O metrics collection.

**Collected metrics when enabled:**
- `disk.read_bytes_per_sec` - Read throughput
- `disk.write_bytes_per_sec` - Write throughput
- `disk.read_ops_per_sec` - Read IOPS
- `disk.write_ops_per_sec` - Write IOPS
- `disk.utilization_percent` - Disk busy percentage

**Disable if:** Disk I/O is never a bottleneck (pure CPU workloads).

---

#### `collectors.disk.devices`

**Type:** List of strings
**Default:** `[]` (empty = all devices)

Limit disk monitoring to specific block devices.

**Examples:**
```yaml
# Monitor all devices (default)
collectors:
  disk:
    devices: []

# Monitor only specific devices
collectors:
  disk:
    devices: ["sda", "nvme0n1"]

# Monitor multiple NVMe drives
collectors:
  disk:
    devices: ["nvme0n1", "nvme1n1", "nvme2n1"]
```

**How to find device names:**
```bash
# List all block devices
lsblk

# Show disk usage
df -h

# Show device stats
iostat -x 1
```

**When to use:**
- Systems with many disks (reduce noise)
- Network-attached storage (exclude slow remote mounts)
- Temporary devices (exclude USB drives, loop devices)

---

#### `collectors.network.enabled`

**Type:** Boolean
**Default:** `true`

Enable or disable network metrics collection.

**Collected metrics when enabled:**
- `network.receive_bytes_per_sec` - Inbound bandwidth
- `network.transmit_bytes_per_sec` - Outbound bandwidth
- `network.receive_packets_per_sec` - Inbound packet rate
- `network.transmit_packets_per_sec` - Outbound packet rate
- `network.receive_errors` - Receive errors (cumulative)
- `network.transmit_errors` - Transmit errors (cumulative)

**Disable if:** Network is never saturated, or not relevant to sizing.

---

#### `collectors.network.interfaces`

**Type:** List of strings
**Default:** `[]` (empty = all interfaces)

Limit network monitoring to specific interfaces.

**Examples:**
```yaml
# Monitor all interfaces (default)
collectors:
  network:
    interfaces: []

# Monitor only primary interface
collectors:
  network:
    interfaces: ["eth0"]

# Monitor multiple interfaces
collectors:
  network:
    interfaces: ["eth0", "eth1", "bond0"]
```

**How to find interface names:**
```bash
# List all interfaces
ip addr

# Show interface statistics
ip -s link

# Network usage
ifstat
```

**When to use:**
- Exclude loopback (`lo`) and docker bridges
- Focus on primary network interfaces
- Exclude VPN tunnels or temporary interfaces

---

#### `collectors.container.enabled`

**Type:** Boolean
**Default:** `true`

Enable or disable container metrics collection via cgroup v2.

**Requirements:**
- Docker installed and running
- cgroup v2 enabled (standard on Ubuntu 22.04+)
- Read access to `/var/lib/docker/` (for container names)
- Read access to `/sys/fs/cgroup/` (for resource usage)

**Collected metrics when enabled:**
- `container.cpu.usage_percent` - Per-container CPU usage
- `container.memory.usage_bytes` - Per-container memory usage
- `container.memory.limit_bytes` - Container memory limits
- `container.io.read_bytes_per_sec` - Container disk read rate
- `container.io.write_bytes_per_sec` - Container disk write rate

**Disable if:**
- No Docker containers running
- Containers not relevant to sizing decisions
- Permission issues accessing Docker data

**Note:** If disabled, host-level analysis still works fine. You just won't get container breakdown.

---

## Common Configuration Scenarios

### Scenario 1: Minimal Resource Usage

**Use case:** Low-resource system, only need basic sizing data
```yaml
collection:
  interval: 30s          # Less frequent collection
  retention_days: 7      # Keep only 1 week

storage:
  path: /var/lib/instvisor/metrics.db

collectors:
  cpu:
    enabled: true
    per_core: false      # Only aggregate CPU
  memory:
    enabled: true
    include_swap: false  # Skip swap metrics
  disk:
    enabled: false       # Disk not important
  network:
    enabled: false       # Network not important
  container:
    enabled: true        # Keep containers
```

**Result:** ~500MB storage, minimal overhead

---

### Scenario 2: Standard Production

**Use case:** Typical production server, balanced monitoring
```yaml
collection:
  interval: 15s
  retention_days: 90

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
    devices: []          # All devices
  network:
    enabled: true
    interfaces: []       # All interfaces
  container:
    enabled: true
```

**Result:** ~5GB storage, comprehensive monitoring

---

### Scenario 3: High-Frequency Monitoring

**Use case:** Debugging performance issues, need detailed data
```yaml
collection:
  interval: 5s           # High frequency
  retention_days: 7      # Keep only recent data

storage:
  path: /data/instvisor/metrics.db  # Fast SSD

collectors:
  cpu:
    enabled: true
    per_core: true       # See per-core bottlenecks
  memory:
    enabled: true
    include_swap: true
  disk:
    enabled: true
    devices: ["nvme0n1"] # Focus on main disk
  network:
    enabled: true
    interfaces: ["eth0"] # Focus on main interface
  container:
    enabled: true
```

**Result:** ~1.5GB storage for 7 days, catches brief spikes

---

### Scenario 4: Long-Term Trend Analysis

**Use case:** Capacity planning, seasonal patterns
```yaml
collection:
  interval: 1m           # Less granular but longer history
  retention_days: 365    # One year of data

storage:
  path: /data/instvisor/metrics.db

collectors:
  cpu:
    enabled: true
    per_core: false      # Aggregate only for space
  memory:
    enabled: true
    include_swap: true
  disk:
    enabled: true
    devices: []
  network:
    enabled: true
    interfaces: []
  container:
    enabled: true
```

**Result:** ~15GB storage for full year

---

### Scenario 5: Container-Only Monitoring

**Use case:** Kubernetes node, only care about container resource usage
```yaml
collection:
  interval: 15s
  retention_days: 30

storage:
  path: /var/lib/instvisor/metrics.db

collectors:
  cpu:
    enabled: true
    per_core: false      # Aggregate sufficient
  memory:
    enabled: true
    include_swap: false  # No swap in K8s
  disk:
    enabled: false       # Volumes handled by K8s
  network:
    enabled: false       # Not sizing-critical
  container:
    enabled: true        # Main focus
```

**Result:** ~2GB storage, container-focused

---

## Changing Configuration

### 1. Edit Configuration File
```bash
# For systemd installation
sudo nano /etc/instvisor/agent.yaml

# For Docker (use volume mount or docker exec)
docker exec -it instvisor nano /etc/instvisor/agent.yaml
```

### 2. Validate Configuration

Before restarting, validate YAML syntax:
```bash
# Check for YAML syntax errors
yamllint /etc/instvisor/agent.yaml

# Or use python
python3 -c "import yaml; yaml.safe_load(open('/etc/instvisor/agent.yaml'))"
```

### 3. Restart Instvisor
```bash
# Systemd
sudo systemctl restart instvisor
sudo systemctl status instvisor

# Docker
docker restart instvisor
docker logs instvisor
```

### 4. Verify Changes

Check logs to confirm new settings:
```bash
# Systemd
sudo journalctl -u instvisor -n 50

# Docker
docker logs --tail 50 instvisor
```

Look for:
```
Starting collector manager with X collectors
Starting collector: cpu (interval: 30s)  ← New interval
```

---

## Configuration Best Practices

### 1. Start with Defaults

Default configuration is optimized for most use cases. Only change when you have a specific reason.

### 2. Match Interval to Workload Variability

- **Highly variable workloads**: 10-15s interval
- **Steady workloads**: 30s-1m interval
- **Batch jobs**: 1m interval sufficient

### 3. Set Retention Based on Analysis Frequency

- **Weekly reviews**: 7-14 days retention
- **Monthly reviews**: 30-60 days retention
- **Quarterly reviews**: 90-120 days retention

### 4. Balance Granularity vs Storage

Remember: `interval × retention = storage cost`

Halving the interval doubles the storage. Doubling retention doubles the storage.

### 5. Use Specific Device/Interface Lists on Large Systems

On systems with 10+ disks or network interfaces, explicitly list what matters to reduce noise.

### 6. Monitor Configuration Changes

After changing configuration, verify:
```bash
# Check metrics count is as expected
sudo sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"

# Check collection is happening
sudo journalctl -u instvisor -f
```

---

## Environment Variables

Instvisor also supports environment variable overrides for container deployments:
```bash
# Override database path
INSTVISOR_DB_PATH=/custom/path/metrics.db

# Override collection interval
INSTVISOR_INTERVAL=30s

# Override retention
INSTVISOR_RETENTION=30
```

**Docker example:**
```bash
docker run -d \
  -e INSTVISOR_INTERVAL=30s \
  -e INSTVISOR_RETENTION=30 \
  abhishekkarki/instvisor:latest
```

**Note:** Environment variables take precedence over config file values.

---

## Troubleshooting Configuration Issues

### Issue: Changes not taking effect

**Solution:** Restart instvisor after configuration changes.
```bash
sudo systemctl restart instvisor
```

### Issue: Invalid YAML syntax

**Error:** `yaml: line X: mapping values are not allowed in this context`

**Solution:** Check indentation (use spaces, not tabs), ensure colons have space after them.
```yaml
# Wrong
collection:interval: 15s

# Correct
collection:
  interval: 15s
```

### Issue: Collector not working after enabling

**Solution:** Check logs for permission errors or missing dependencies:
```bash
sudo journalctl -u instvisor -n 100 | grep -i error
```

### Issue: Database growing too large

**Solution:**
1. Reduce `retention_days`
2. Increase `interval`
3. Disable unnecessary collectors
4. Manually clean old data: `sudo sqlite3 /var/lib/instvisor/metrics.db "DELETE FROM metrics WHERE timestamp < strftime('%s', 'now', '-30 days');"`

---

## Next Steps

- [Getting Started Guide](getting-started.md) - Installation and first analysis
- [Understanding Recommendations](recommendations.md) - How sizing works
- [Container Metrics Guide](container-metrics.md) - Container monitoring details
- [Troubleshooting](troubleshooting.md) - Common issues

---

**Need help?** Open an issue on [GitHub](https://github.com/abhishekkarki/instvisor/issues) or ask in [Discussions](https://github.com/abhishekkarki/instvisor/discussions).
