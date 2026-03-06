# Getting Started with Instvisor

This guide will help you install and start using instvisor to analyze your system's resource usage and get right-sizing recommendations.

---

## Prerequisites

Before installing instvisor, ensure you have:

### System Requirements
- **Operating System**: Linux with kernel 5.x+ (Ubuntu 22.04+, RHEL 9+, Debian 12+)
- **Architecture**: amd64 (x86_64) or arm64 (aarch64)
- **cgroup version**: cgroup v2 (standard on modern Linux distributions)
- **Permissions**: Root access or sudo privileges
- **Disk Space**: ~100MB for binary and dependencies, ~1-5GB for metrics database (depending on retention)

### Optional (for container metrics)
- **Docker**: Version 20.x or later (if monitoring containers)
- **Access**: Read access to `/var/lib/docker/` and `/sys/fs/cgroup/`

### Check Your System
```bash
# Check cgroup version (should output: cgroup2fs)
stat -fc %T /sys/fs/cgroup/

# Check available disk space
df -h /var/lib

# Check if Docker is running (optional)
docker ps
```

---

## Installation

Choose the installation method that best fits your environment:

### Method 1: Binary Installation (Recommended)

**Step 1: Download the binary**
```bash
# Download latest release
wget https://github.com/abhishekkarki/instvisor/releases/latest/download/instvisor-linux-amd64.tar.gz

# Extract
tar -xzf instvisor-linux-amd64.tar.gz

# Verify contents
ls -la
# Should show: build/, configs/, scripts/, deployments/, README.md, LICENSE
```

**Step 2: Install with systemd**
```bash
# Run installation script (requires sudo)
sudo ./scripts/install.sh
```

This script will:
- Create `instvisor` system user
- Create directories: `/etc/instvisor/`, `/var/lib/instvisor/`
- Copy binaries to `/usr/local/bin/`
- Install systemd service
- Enable auto-start on boot

**Step 3: Start the service**
```bash
# Start instvisor
sudo systemctl start instvisor

# Enable on boot
sudo systemctl enable instvisor

# Check status
sudo systemctl status instvisor
```

**Expected output:**
```
● instvisor.service - Instvisor System Resource Monitor
     Loaded: loaded (/etc/systemd/system/instvisor.service; enabled)
     Active: active (running) since Thu 2026-02-27 10:00:00 UTC
```

---

### Method 2: Docker Installation

**Step 1: Pull the image**
```bash
docker pull abhishekkarki/instvisor:latest
```

**Step 2: Run the container**
```bash
docker run -d \
  --name instvisor \
  --privileged \
  --pid=host \
  -v /:/rootfs:ro \
  -v /sys:/sys:ro \
  -v /var/run:/var/run:ro \
  -v instvisor-data:/var/lib/instvisor \
  abhishekkarki/instvisor:latest
```

**Why these flags?**
- `--privileged`: Access to host devices for metrics collection
- `--pid=host`: See host processes, not just container processes
- `-v /:/rootfs:ro`: Read-only access to host filesystem for `/proc`
- `-v /sys:/sys:ro`: Access to cgroup metrics
- `-v instvisor-data:/var/lib/instvisor`: Persist metrics database

**Step 3: Verify it's running**
```bash
# Check container status
docker ps | grep instvisor

# View logs
docker logs instvisor

# Expected: "Starting collector manager with X collectors"
```

**Using docker-compose:**

Create `docker-compose.yml`:
```yaml
services:
  instvisor:
    image: abhishekkarki/instvisor:latest
    container_name: instvisor
    privileged: true
    pid: host
    volumes:
      - /:/rootfs:ro
      - /sys:/sys:ro
      - /var/run:/var/run:ro
      - instvisor-data:/var/lib/instvisor
    restart: unless-stopped

volumes:
  instvisor-data:
```

Then run:
```bash
docker-compose up -d
```

---

### Method 3: Build from Source

**Prerequisites:**
- Go 1.21 or later
- gcc (for CGO/SQLite)
- make
```bash
# Clone repository
git clone https://github.com/abhishekkarki/instvisor.git
cd instvisor

# Install dependencies
go mod download

# Build binaries
make build-all

# Install (optional)
sudo make install
```

---

## Verify Installation

After installation, verify instvisor is collecting metrics:

### Check Logs

**For systemd:**
```bash
sudo journalctl -u instvisor -f
```

**For Docker:**
```bash
docker logs -f instvisor
```

**Expected output (every 15 seconds):**
```
2026/02/27 10:00:15 Collected and stored 28 metrics from cpu
2026/02/27 10:00:15 Collected and stored 10 metrics from memory
2026/02/27 10:00:15 Collected and stored 12 metrics from disk
2026/02/27 10:00:15 Collected and stored 8 metrics from network
2026/02/27 10:00:15 Collected and stored 45 metrics from container
```

### Check Database

**For systemd:**
```bash
sudo ls -lh /var/lib/instvisor/metrics.db
```

**For Docker:**
```bash
docker exec instvisor ls -lh /var/lib/instvisor/metrics.db
```

**Expected:** File size growing over time (starts at ~32KB, grows to ~1-5GB over weeks)

### Query Metrics Directly

**For systemd:**
```bash
sudo sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"
```

**For Docker:**
```bash
docker exec instvisor sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"
```

**Expected:** Number increasing every 15 seconds (131+ metrics per cycle)

---

## Wait for Data Collection

⏳ **Important:** Instvisor needs time to collect meaningful data.

- **Minimum**: 24 hours (for basic analysis)
- **Recommended**: 7 days (for P95 statistics)
- **Optimal**: 14-30 days (for pattern detection)

**Why?** Percentile calculations need enough samples to be statistically valid. With 15-second intervals:
- 1 day = 5,760 samples
- 7 days = 40,320 samples  (Good for P95)
- 30 days = 172,800 samples (Great for patterns)

---

## Run Your First Analysis

After collecting data for at least 24 hours:

### Binary Installation
```bash
sudo instvisor-analyze -days 7
```

### Docker Installation
```bash
docker exec instvisor instvisor-analyze -days 7
```

### Understanding the Output
```
Current System: my-server - 8 vCPUs, 32.0 GB RAM (linux/amd64)

=== RESOURCE ANALYSIS ===
Analysis Period: 168h0m0s
Workload Pattern: steady_state (confidence: 90%)

CPU Usage:
  Mean:   25.3%
  P95:    42.1%    ← Key metric for sizing
  Max:    68.0%
  Samples: 40,320

Memory Usage:
  Mean:   45.2%
  P95:    52.3%    ← Key metric for sizing
  Max:    58.0%

=== CONTAINER RESOURCE BREAKDOWN ===
Top CPU Consumers:
  1. postgres                        CPU:  35.2%  (84% of host)
  2. nginx                           CPU:   4.1%  (10% of host)

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

**Key sections explained:**

1. **Workload Pattern**: 
   - `steady_state` = predictable, constant load
   - `bursty` = unpredictable spikes
   - `scheduled` = regular patterns (e.g., cron jobs)

2. **P95 vs Mean**:
   - **Mean (25.3%)**: Average usage
   - **P95 (42.1%)**: 95% of the time CPU is below this ← Used for sizing
   - **Max (68.0%)**: Highest spike recorded

3. **Container Breakdown**:
   - Shows which containers drive resource usage
   - Helps decide: optimize container vs scale host

4. **Recommendations**:
   - Based on P95 + 20% headroom (configurable)
   - Maps to real cloud instance types

---

## Common Questions

### Q: Why is P95 used instead of average or max?

**A:** 
- **Average (P50)** = Too low, you'll run out of resources during normal peaks
- **Max** = Too high, you'll pay for capacity needed once a month
- **P95** = Sweet spot - handles 95% of situations, allows occasional spikes

Think of it like this: Size for normal peaks, not absolute worst case.

### Q: I just installed, why does analysis show "No data found"?

**A:** You need at least 2 collection cycles (30 seconds minimum) for rate-based metrics like CPU and disk I/O. For meaningful analysis, wait 24 hours minimum, 7 days recommended.

### Q: What if I don't have Docker containers?

**A:** Instvisor works perfectly fine without containers! You'll get host-level analysis and recommendations. The container section simply won't appear.

### Q: How much disk space will instvisor use?

**A:** Approximately:
- **7 days retention**: ~500MB
- **30 days retention**: ~2GB
- **90 days retention**: ~5GB

Calculation: `(metrics_per_cycle × bytes_per_metric × cycles_per_day × retention_days)`

### Q: Can I run analysis more frequently?

**A:** Yes, but keep in mind:
- Analysis is a snapshot of a time window
- You can run it hourly, daily, weekly - doesn't affect collection
- More frequent analysis doesn't give better recommendations, just more frequent checks

### Q: What if recommendations seem wrong?

**A:** Check these:
1. **Collection period too short**: Need 7+ days for valid P95
2. **Recent spike**: A one-time event skewed the P95
3. **Seasonal pattern**: Weekly/monthly patterns need longer collection
4. **Workload changed**: Old data may not reflect current state

---

## Next Steps

Now that instvisor is running:

1. **Wait 7 days** for optimal data collection
2. **Run weekly analysis** to track trends
3. **Act on recommendations**:
   - Investigate high-usage containers first
   - Optimize before scaling
   - Test smaller instance sizes in staging
4. **Calculate ROI**: Current cost vs recommended cost
5. **Customize configuration** if needed (see [Configuration Guide](configuration.md))

---

## Troubleshooting

### Issue: Service won't start
```bash
# Check logs for errors
sudo journalctl -u instvisor -n 50

# Common causes:
# - Permission denied on /proc or /sys
# - Database path not writable
# - Port conflict (if exporter enabled)
```

**Solution:** Ensure instvisor runs as root or has CAP_SYS_ADMIN capability.

### Issue: No container metrics
```bash
# Check Docker is running
docker ps

# Check permissions
sudo ls -l /var/lib/docker/containers/

# Check cgroup v2
stat -fc %T /sys/fs/cgroup/
```

**Solution:** 
- Ensure `/var/lib/docker/` is readable
- Verify cgroup v2 is enabled
- On Docker, run with `-v /var/lib/docker:/var/lib/docker:ro`

### Issue: Database growing too large
```bash
# Check current size
sudo du -h /var/lib/instvisor/metrics.db

# Reduce retention (edit config)
sudo nano /etc/instvisor/agent.yaml
# Change: retention_days: 30  (from 90)

# Restart
sudo systemctl restart instvisor
```

### Issue: Analysis is slow

**Cause:** Large database with millions of rows.

**Solution:**
- Reduce retention period
- Increase collection interval (15s → 30s)
- Run analysis on specific metrics only (future feature)

---

## Getting Help

- [Configuration Reference](configuration.md)
- [Understanding Recommendations](recommendations.md)
- [Container Metrics Guide](container-metrics.md)
- [Troubleshooting Guide](troubleshooting.md)
- [GitHub Discussions](https://github.com/abhishekkarki/instvisor/discussions)
- [Report Issues](https://github.com/abhishekkarki/instvisor/issues)

---

**Ready to optimize your infrastructure? Let instvisor guide you!** 