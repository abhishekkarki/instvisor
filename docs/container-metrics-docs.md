# Container Metrics Guide


This guide explains how instvisor collects and analyzes container-level resource metrics.


---


## Overview


Instvisor monitors Docker containers by reading **cgroup v2** (control groups) - the Linux kernel mechanism that Docker uses to track and limit container resources.


**Key benefit:** You get per-container resource usage without installing agents inside containers or using the Docker API.


---


## How Container Metrics Work


### The Big Picture
```
Docker Container
   ↓
Linux Kernel creates cgroup
   ↓
Kernel writes resource usage to /sys/fs/cgroup/
   ↓
Instvisor reads these files
   ↓
Calculates per-container metrics
   ↓
Stores in SQLite with container labels
```


### What Are cgroups?


**cgroups (control groups)** are a Linux kernel feature that:
- Track resource usage per process group
- Enforce resource limits (CPU, memory, I/O)
- Provide isolation between containers


When Docker starts a container, the kernel automatically:
1. Creates a cgroup for that container
2. Puts all container processes in that cgroup
3. Tracks all resource usage in real-time
4. Writes statistics to files in `/sys/fs/cgroup/`


**Think of it like this:**
```
Host OS = Building
cgroups = Apartment meters
Each container = Apartment with its own electricity/water meter
```


You can see total building usage (host metrics) AND per-apartment usage (container metrics).


---


## cgroup v2 vs v1


### cgroup v2 (Modern - What Instvisor Uses)


**Path structure:**
```
/sys/fs/cgroup/system.slice/docker-<container-id>.scope/
   ├── cpu.stat          # CPU usage
   ├── memory.current    # Current memory usage
   ├── memory.max        # Memory limit
   └── io.stat           # Disk I/O
```


**Supported on:**
- Ubuntu 22.04+ (default)
- Debian 12+ (default)
- RHEL 9+ (default)
- Fedora 31+ (default)


**Check if you have cgroup v2:**
```bash
stat -fc %T /sys/fs/cgroup/


# Output should be: cgroup2fs
```


### cgroup v1 (Legacy)


**Path structure:**
```
/sys/fs/cgroup/cpu/docker/<container-id>/
/sys/fs/cgroup/memory/docker/<container-id>/
/sys/fs/cgroup/blkio/docker/<container-id>/
```


**Instvisor currently does NOT support cgroup v1.** If you're on an older system, consider:
- Upgrading to a modern Linux distribution
- Manually enabling cgroup v2 (see troubleshooting section)


---


## What Metrics Are Collected


### CPU Metrics


#### `container.cpu.usage_percent`


**What it measures:** Percentage of host CPU this container is using.


**How it's calculated:**
```
1. Read cumulative CPU time from cgroup:
  /sys/fs/cgroup/.../docker-<id>.scope/cpu.stat
 
  usage_usec 192042081  # Microseconds of CPU time used since container start


2. Calculate delta from last reading:
  usage_diff = current_usage - last_usage
  time_diff = current_time - last_time


3. Calculate percentage:
  cpu_percent = (usage_diff / time_diff) × 100
```


**Example:**
```
Reading at 10:00:00: usage_usec = 100,000,000 (100 seconds)
Reading at 10:00:15: usage_usec = 102,500,000 (102.5 seconds)


Delta: 2,500,000 microseconds = 2.5 seconds of CPU time
Time elapsed: 15 seconds


CPU usage = (2.5 / 15) × 100 = 16.67%
```


**Important notes:**
- This is relative to **one CPU core**
- If a container uses 200%, it's using 2 full cores
- Multi-core containers will show >100% usage


**Labels:**
```json
{
 "container_id": "27995b46e30e",
 "container_name": "postgres",
 "image": "postgres:15"
}
```


---


#### `container.cpu.usage_total_usec`


**What it measures:** Cumulative CPU time used since container started (in microseconds).


**Use case:** Tracking total CPU consumption over container lifetime.


**Example:**
```
Container started: Monday 9am
Current time: Friday 5pm (5 days later)


usage_total_usec: 432,000,000 microseconds
               = 432 seconds
               = 7.2 minutes of CPU time


Average CPU usage: 7.2 minutes / (5 days × 24 hours × 60 minutes)
                = 7.2 / 7200 = 0.1% average CPU
```


---


#### `container.cpu.user_usec` and `container.cpu.system_usec`


**What they measure:**
- **user_usec**: Time spent running application code
- **system_usec**: Time spent in kernel (system calls, I/O)


**Use case:** Identify if container is CPU-bound (high user) or I/O-bound (high system).


**Example:**
```
Container CPU breakdown:
 Total:  50% CPU
 User:   45% (application code)
 System:  5% (kernel/I/O)


Interpretation: CPU-bound workload (mostly application logic)


Container CPU breakdown:
 Total:  50% CPU
 User:   10% (application code)
 System: 40% (kernel/I/O)


Interpretation: I/O-bound workload (lots of disk/network operations)
```


---


### Memory Metrics


#### `container.memory.usage_bytes`


**What it measures:** Current memory usage by the container in bytes.


**How it's read:**
```bash
cat /sys/fs/cgroup/system.slice/docker-<id>.scope/memory.current


# Output: 1873158144  (bytes)
#       = 1.74 GB
```


**What's included:**
- Application memory (heap, stack)
- Page cache (file system cache)
- Buffers
- Shared memory


**Important:** This includes **cache**, which Linux can reclaim. Real memory pressure is lower than this number suggests.


**Example:**
```
memory.current: 2 GB


Breakdown:
 Application RSS:     1.2 GB  (actual app memory)
 Page cache:          0.6 GB  (can be freed)
 Buffers:             0.2 GB  (can be freed)


Real usage: ~1.2 GB
Reported usage: 2 GB
```


---


#### `container.memory.limit_bytes`


**What it measures:** Memory limit set for the container (if any).


**How it's read:**
```bash
cat /sys/fs/cgroup/system.slice/docker-<id>.scope/memory.max


# Output: 4294967296  (4 GB limit)
# Or:     max          (no limit)
```


**Use case:** Check if container is constrained by limits.


**Example with Docker:**
```bash
# Container with 2GB limit
docker run -m 2g postgres


# memory.max shows: 2147483648 (2 GB)
# memory.current shows: 1800000000 (1.8 GB)
# Usage: 90% of limit - close to OOM!
```


---


#### `container.memory.usage_percent`


**What it measures:** Percentage of memory limit being used (only if limit is set).


**Calculation:**
```
usage_percent = (memory.current / memory.max) × 100
```


**Important:** Only available if container has a memory limit. If `memory.max = max` (unlimited), this metric is not collected.


**Example:**
```
With limit (Docker: -m 4g):
 memory.current: 3 GB
 memory.max: 4 GB
 usage_percent: 75%


Without limit:
 memory.current: 3 GB
 memory.max: max
 usage_percent: (not calculated)
```


---


### I/O Metrics


#### `container.io.read_bytes_per_sec` and `container.io.write_bytes_per_sec`


**What they measure:** Disk throughput (bytes read/written per second).


**How it's calculated:**
```
1. Read cumulative I/O from cgroup:
  /sys/fs/cgroup/.../docker-<id>.scope/io.stat
 
  252:0 rbytes=282308608 wbytes=13915918336


2. Calculate delta:
  read_diff = current_rbytes - last_rbytes
  write_diff = current_wbytes - last_wbytes
  time_diff = 15 seconds (collection interval)


3. Calculate rate:
  read_bytes_per_sec = read_diff / time_diff
  write_bytes_per_sec = write_diff / time_diff
```


**Example:**
```
Time 1: rbytes=100,000,000 (100 MB)
Time 2: rbytes=115,000,000 (115 MB) [15 seconds later]


Diff: 15 MB in 15 seconds = 1 MB/sec read throughput
```


**Labels include device:**
```json
{
 "container_id": "27995b46e30e",
 "container_name": "postgres",
 "device": "252:0"  # Major:Minor device number
}
```


**Convert device number to name:**
```bash
# Find device by major:minor
ls -l /dev | grep "252,.*0"
# Output: brw-rw---- 1 root disk 252, 0 Feb 27 10:00 sda
```


---


#### `container.io.read_ops_per_sec` and `container.io.write_ops_per_sec`


**What they measure:** I/O operations per second (IOPS).


**Use case:** Identify I/O-intensive containers.


**Example:**
```
Container A (database):
 write_bytes_per_sec: 5 MB/sec
 write_ops_per_sec: 500 IOPS
 Average write size: 5 MB / 500 = 10 KB (many small writes - typical DB)


Container B (backup):
 write_bytes_per_sec: 50 MB/sec
 write_ops_per_sec: 50 IOPS
 Average write size: 50 MB / 50 = 1 MB (few large writes - sequential)
```


---


#### `container.io.read_bytes_total` and `container.io.write_bytes_total`


**What they measure:** Cumulative bytes read/written since container started.


**Use case:** Understand total I/O volume over container lifetime.


**Example:**
```
Container running for 7 days:
 write_bytes_total: 500 GB
 Average daily writes: 500 GB / 7 = 71 GB/day


Planning: If this container is writing 71 GB/day, ensure:
- Disk has enough space
- Backup strategy accounts for this volume
- I/O limits won't throttle this workload
```


---


## Container Identification


### How Instvisor Gets Container Names


Instvisor needs to map container IDs to human-readable names. It does this by reading Docker's config files.


**Step 1: Find containers**
```bash
# List all docker cgroups
ls /sys/fs/cgroup/system.slice/docker-*.scope


# Example output:
/sys/fs/cgroup/system.slice/docker-27995b46e30e7df60eefcfade074faf57410dde7c5a412906e3d40d29976c278.scope
```


**Step 2: Extract container ID**
```
Filename: docker-27995b46e30e7df60eefcfade074faf57410dde7c5a412906e3d40d29976c278.scope
Container ID: 27995b46e30e7df60eefcfade074faf57410dde7c5a412906e3d40d29976c278
Short ID: 27995b46e30e (first 12 chars)
```


**Step 3: Read Docker config**
```bash
# Docker stores metadata here
cat /var/lib/docker/containers/27995b46e30e.../config.v2.json


# Extract name and image:
{
 "Name": "/instvisor",
 "Config": {
   "Image": "abhishekkarki/instvisor:latest"
 }
}
```


**Step 4: Store in metrics**
```
Metric: container.cpu.usage_percent
Value: 15.3
Labels:
 container_id: 27995b46e30e
 container_name: instvisor
 image: abhishekkarki/instvisor:latest
```


### Fallback Behavior


If instvisor **cannot read** `/var/lib/docker/` (permissions issue), it falls back to:
```json
{
 "container_id": "27995b46e30e",
 "container_name": "27995b46e30e",  // Uses ID as name
 "image": "unknown"
}
```


**Still works!** You get metrics with container IDs instead of names.


---


## Container Breakdown in Analysis


### How Container Analysis Works


When you run `instvisor-analyze`, it:


1. **Queries container metrics** from SQLite
2. **Groups by container** (using container_id label)
3. **Calculates P95** for each container
4. **Compares to host** P95 to get percentage


**Example query flow:**
```sql
-- Get all CPU metrics for container 'postgres'
SELECT value FROM metrics
WHERE metric_name = 'container.cpu.usage_percent'
 AND json_extract(labels, '$.container_name') = 'postgres'
 AND timestamp >= <start>
 AND timestamp <= <end>;


-- Returns 40,320 values (7 days × 15s interval)


-- Calculate P95
-- Sort values, take value at position 38,304 (95% of 40,320)
```


### Percentage of Host Calculation
```
Host CPU P95: 70%
Container 'postgres' P95: 49%


Percentage of host = (49 / 70) × 100 = 70%


Display:
 postgres: 49% CPU (70% of host)
```


**Interpretation:** Postgres drives 70% of your CPU requirements. If you optimize postgres, you could potentially downsize the host significantly.


---


## Understanding Container Insights


### Threshold: Dominates (>60% of host)


**Trigger:**
```
Container uses >60% of host P95 resources
```


**Insight displayed:**
```
  Container 'postgres' consumes 70% of your host CPU
  Consider optimizing 'postgres' before scaling the entire host
```


**Why 60%?**
- A single container driving >60% means it's the bottleneck
- Scaling the host might not help if the container can't use more resources
- Optimization is more cost-effective than scaling


**Actions to take:**
1. **Database containers (postgres, mysql, mongodb):**
  - Analyze slow queries: `EXPLAIN ANALYZE`
  - Add missing indexes
  - Tune connection pooling
  - Enable query caching
  - Consider read replicas


2. **Application containers:**
  - Profile the application (CPU profiling)
  - Check for infinite loops or inefficient algorithms
  - Optimize hot code paths
  - Add caching layers


3. **Alternative: Separate the container**
  - Move heavy container to dedicated host/instance
  - Size each independently
  - Example: Move postgres to managed RDS


---


### Threshold: Top Consumer (40-60% of host)


**Trigger:**
```
Container uses 40-60% of host P95 resources
```


**Insight displayed:**
```
Container 'nginx' is your largest CPU consumer (55% of host)
```


**Interpretation:** This container is the primary consumer but not completely dominating. Worth investigating but not urgent.


**Actions:**
- Review container configuration
- Check if resource usage is expected for this workload
- Consider if this container could be optimized


---


### Threshold: Distributed (<30% per container)


**Trigger:**
```
No single container uses >30% of host
```


**Insight displayed:**
```
CPU usage is well-distributed. Top consumer: 'api' (25% of host)
Resource usage is evenly distributed across containers - no single bottleneck
```


**Interpretation:** This is **ideal** - no single container drives sizing. Right-sizing the host benefits all containers proportionally.


**Actions:**
- Trust the host-level recommendation
- No container-specific optimization needed
- Downsizing the host is safe


---


## Real-World Examples


### Example 1: PostgreSQL Database Container


**Scenario:**
```
Host: 8 vCPUs, 32 GB RAM
Container: postgres


Analysis output:
=== CONTAINER BREAKDOWN ===
 postgres: 56% CPU (80% of host)


=== INSIGHTS ===
Container 'postgres' consumes 80% of your host CPU
```


**Investigation:**
```sql
-- Check slow queries
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
ORDER BY total_time DESC
LIMIT 10;


-- Found: Missing index on frequently joined column
```


**Action taken:**
```sql
CREATE INDEX idx_users_email ON users(email);
```


**Result after 7 days:**
```
postgres: 28% CPU (40% of host)
Host P95 dropped from 70% → 35%
Recommendation changed: 8 vCPUs → 4 vCPUs
Cost savings: 50% (€245/month → €122/month)
```


**Key lesson:** Optimization saved more than scaling up ever could.


---


### Example 2: Microservices Platform


**Scenario:**
```
Host: 4 vCPUs, 16 GB RAM
7 containers running (API, auth, payments, notifications, etc.)


Analysis output:
=== CONTAINER BREAKDOWN ===
Top CPU Consumers:
 1. api-gateway      15% (21% of host)
 2. payment-service  12% (17% of host)
 3. auth-service     10% (14% of host)
 4. notifications     8% (11% of host)
 5. analytics         7% (10% of host)


=== INSIGHTS ===
💡 Resource usage is evenly distributed across containers
```


**Interpretation:** No bottleneck, well-architected microservices.


**Recommendation followed:** Downsize from 4 vCPUs to 2 vCPUs.


**Result:**
- All services still performant
- 50% cost reduction
- No single service impacted


**Key lesson:** Distributed load = safe to downsize.


---


### Example 3: RabbitMQ Message Queue


**Scenario:**
```
Host: 8 vCPUs, 32 GB RAM
Container: rabbitmq


Analysis output:
=== CONTAINER BREAKDOWN ===
 rabbitmq: 6% CPU, 12 GB memory (37% of host memory)


=== INSIGHTS ===
Container 'rabbitmq' uses 37% of host memory
```


**Investigation:** Memory usage high but expected for message queuing (messages in RAM).


**Decision:** Don't downsize memory, but can reduce vCPUs.


**Recommendation followed:**
```
From: 8 vCPUs, 32 GB
To:   4 vCPUs, 32 GB  (keep memory, reduce CPU)
```


**Result:** €60/month savings, no performance impact.


**Key lesson:** CPU and memory can be sized independently based on container needs.


---


## Troubleshooting


### Issue: No container metrics collected


**Symptom:**
```
=== CONTAINER RESOURCE BREAKDOWN ===
No container metrics available
```


**Diagnosis:**
```bash
# 1. Check if Docker is running
docker ps


# 2. Check cgroup v2 is enabled
stat -fc %T /sys/fs/cgroup/
# Should output: cgroup2fs


# 3. Check if cgroups exist
ls /sys/fs/cgroup/system.slice/docker-*.scope


# 4. Check permissions
ls -ld /sys/fs/cgroup/system.slice/
ls -ld /var/lib/docker/containers/
```


**Solutions:**


**A) Docker not running:**
```bash
sudo systemctl start docker
sudo systemctl enable docker
```


**B) cgroup v1 (not supported):**
```bash
# Check current version
mount | grep cgroup


# If cgroup v1, enable v2:
# Add to /etc/default/grub:
GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=1"


# Update grub and reboot
sudo update-grub
sudo reboot
```


**C) Permission denied on /var/lib/docker:**
```bash
# For systemd installation
sudo chmod o+rx /var/lib/docker/containers/


# For Docker installation, mount it:
docker run -v /var/lib/docker:/var/lib/docker:ro ...
```


---


### Issue: Container names show as IDs


**Symptom:**
```
=== CONTAINER BREAKDOWN ===
 27995b46e30e: 15% CPU  # Shows ID instead of name
```


**Diagnosis:**
```bash
# Check if Docker config is readable
ls -la /var/lib/docker/containers/27995b46e30e*/config.v2.json
```


**Cause:** Permission denied on `/var/lib/docker/containers/`.


**Solution:**
```bash
# Grant read access (safe - configs are not sensitive)
sudo chmod -R o+rX /var/lib/docker/containers/


# Or for Docker deployment:
docker run -v /var/lib/docker:/var/lib/docker:ro ...
```


**Workaround:** Metrics still work! Just shown with IDs instead of names. Use `docker ps` to map IDs to names manually.


---


### Issue: Memory usage seems too high


**Symptom:**
```
Container 'nginx': 2 GB memory usage
But docker stats shows: 500 MB
```


**Explanation:** `memory.current` includes page cache (file system cache), which Linux can reclaim.


**To see "real" usage:**
```bash
# Check memory breakdown
cat /sys/fs/cgroup/system.slice/docker-<id>.scope/memory.stat


# Look for:
anon 524288000        # Anonymous memory (RSS) - actual app usage
file 1500000000       # File cache - can be freed
```


**Not a bug:** instvisor shows total memory.current (including cache) because that's what the kernel tracks. Cache can be freed, but it's still using RAM.


---


### Issue: I/O metrics are zero


**Symptom:**
```
container.io.read_bytes_per_sec: 0
container.io.write_bytes_per_sec: 0
```


**Causes:**


**A) Container using tmpfs/volume:**
```bash
# Check container mounts
docker inspect <container> | grep Mounts -A 20


# tmpfs doesn't show in io.stat
```


**B) I/O controller not enabled:**
```bash
# Check if I/O tracking is enabled
cat /sys/fs/cgroup/system.slice/docker-<id>.scope/io.stat


# If empty, I/O controller may not be enabled
```


**Solution:** Enable I/O controller (requires kernel config, advanced).


---


## Best Practices


### 1. Give Instvisor Access to Docker Configs


For human-readable container names:
```bash
# Systemd installation
sudo chmod o+rX /var/lib/docker/containers/


# Docker installation
docker run -v /var/lib/docker:/var/lib/docker:ro ...
```


### 2. Set Memory Limits on Containers


To get `container.memory.usage_percent`:
```bash
# Set limit when running
docker run -m 2g postgres


# Or in docker-compose.yml
services:
 postgres:
   image: postgres:15
   deploy:
     resources:
       limits:
         memory: 2G
```


### 3. Use Meaningful Container Names


Instead of auto-generated names:
```bash
# Bad (auto-generated name)
docker run postgres  # Name: silly_goldberg


# Good (explicit name)
docker run --name postgres-prod postgres
```


### 4. Tag Container Images


For better visibility in metrics:
```bash
# Bad
image: postgres


# Good
image: postgres:15.2
```


### 5. Monitor Container Metrics Alongside Host


Don't just look at host metrics - always check container breakdown:
```bash
# Full analysis
instvisor-analyze -days 7


# Check both:
# - Host CPU P95
# - Container breakdown
# - Container insights
```


---


## Advanced: Container Metrics for Kubernetes


**Note:** Instvisor currently supports Docker directly. For Kubernetes:


### Option 1: Run as DaemonSet


Deploy instvisor on each node:
```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
 name: instvisor
spec:
 template:
   spec:
     hostPID: true
     containers:
     - name: instvisor
       image: abhishekkarki/instvisor:latest
       securityContext:
         privileged: true
       volumeMounts:
       - name: sys
         mountPath: /sys
         readOnly: true
       - name: docker
         mountPath: /var/lib/docker
         readOnly: true
     volumes:
     - name: sys
       hostPath:
         path: /sys
     - name: docker
       hostPath:
         path: /var/lib/docker
```


Each instvisor instance analyzes its node's containers.


### Option 2: Pod-Level Recommendations (Future)


Coming soon: Direct pod resource recommendations for Kubernetes.


See [Issue #XX](https://github.com/abhishekkarki/instvisor/issues/XX) for roadmap.


---


## Comparison with Other Tools


| Tool | Container Metrics | Per-Container Analysis | Recommendations |
|------|-------------------|------------------------|-----------------|
| **instvisor** | Yes (cgroup v2) |  Yes |  Yes |
| **cAdvisor** |  Yes |  No |  No |
| **docker stats** |  Yes (real-time only) |  No |  No |
| **Prometheus + cAdvisor** |  Yes | Manual (PromQL) |  No |
| **Datadog/New Relic** |  Yes |  Yes |  Limited |


**Instvisor's advantage:** Combines collection + analysis + recommendations in one tool, with no external dependencies.


---


## FAQ


### Q: Does instvisor slow down containers?


**A:** No. Instvisor only **reads** cgroup files created by the kernel. It doesn't interfere with container execution.


**Overhead:** <1% CPU, <50MB RAM for instvisor itself.


### Q: Can I monitor containers without Docker?


**A:** Theoretically yes (Podman, containerd), but instvisor currently assumes Docker's cgroup naming scheme. Support for other runtimes is on the roadmap.


### Q: What about Windows containers?


**A:** Not supported. Instvisor requires Linux cgroups, which don't exist on Windows. Windows Server containers use different mechanisms.


### Q: Can I get network metrics per container?


**A:** Not yet. cgroup v2 doesn't expose network stats (only CPU, memory, I/O). Network monitoring requires other approaches (eBPF, packet capture).


### Q: How accurate are container metrics?


**A:** Very accurate - they come directly from the kernel's accounting. Same data Docker uses for `docker stats`.


### Q: What if a container stops/restarts?


**A:** Historical data is preserved in SQLite (tagged with container_id). New container with same name gets different ID, treated as separate entity.


---


## Summary


**Key Takeaways:**


1. **Container metrics use cgroups** - kernel-level tracking, no agents needed
2. **cgroup v2 required** - modern Linux distributions have this by default
3. **Container breakdown is powerful** - shows which containers drive sizing
4. **Optimization > Scaling** - fix heavy containers before resizing host
5. **Distributed load is ideal** - means right-sizing helps all containers


**Container insights decision tree:**
```
Container using >60% of host? → Yes → Optimize container first
                              ↓ No
Container using 40-60%?        → Yes → Review, likely okay
                              ↓ No
Load distributed (<30% each)?  → Yes → Safe to right-size host
```


---


## Related Documentation


- [Getting Started Guide](getting-started.md) - Installation
- [Configuration Reference](configuration.md) - Enable/disable container collector
- [Understanding Recommendations](recommendations.md) - How container insights affect sizing
- [Troubleshooting](troubleshooting.md) - Container-specific issues


---


**Questions?** Open an issue on [GitHub](https://github.com/abhishekkarki/instvisor/issues) or discuss in [Discussions](https://github.com/abhishekkarki/instvisor/discussions).
