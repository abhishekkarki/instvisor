# Troubleshooting Guide

This guide helps you diagnose and fix common issues with instvisor.

---

## Table of Contents

- [Installation Issues](#installation-issues)
- [Service/Agent Issues](#serviceagent-issues)
- [No Metrics Collected](#no-metrics-collected)
- [Container Metrics Not Working](#container-metrics-not-working)
- [Analysis Issues](#analysis-issues)
- [Performance Issues](#performance-issues)
- [Database Issues](#database-issues)
- [Permission Issues](#permission-issues)
- [Getting Help](#getting-help)

---

## Quick Diagnostic Commands

Run these first to gather system information:
```bash
# Check instvisor status
sudo systemctl status instvisor
# Or for Docker:
docker ps | grep instvisor

# Check logs
sudo journalctl -u instvisor -n 100 --no-pager
# Or for Docker:
docker logs --tail 100 instvisor

# Check database exists
ls -lh /var/lib/instvisor/metrics.db
# Or for Docker:
docker exec instvisor ls -lh /var/lib/instvisor/metrics.db

# Count metrics in database
sudo sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"
# Or for Docker:
docker exec instvisor sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"

# Check cgroup version
stat -fc %T /sys/fs/cgroup/

# Check Docker running
docker ps
```

---

## Installation Issues

### Issue: Binary not found in PATH

**Symptom:**
```bash
$ instvisor-agent
bash: instvisor-agent: command not found
```

**Diagnosis:**
```bash
# Check where binary was installed
which instvisor-agent

# Check installation directory
ls -la /usr/local/bin/instvisor-*
```

**Solution:**

**Option A: Add to PATH**
```bash
export PATH=$PATH:/usr/local/bin
echo 'export PATH=$PATH:/usr/local/bin' >> ~/.bashrc
source ~/.bashrc
```

**Option B: Reinstall**
```bash
cd ~/instvisor
sudo ./scripts/install.sh
```

**Option C: Run with full path**
```bash
/usr/local/bin/instvisor-agent -config /etc/instvisor/agent.yaml
```

---

### Issue: Installation script fails

**Symptom:**
```bash
$ sudo ./scripts/install.sh
Error: Failed to create user 'instvisor'
```

**Diagnosis:**
```bash
# Check if user already exists
id instvisor

# Check script permissions
ls -la scripts/install.sh
```

**Solutions:**

**A) User already exists (harmless):**
```bash
# Skip user creation, just copy files
sudo cp build/instvisor-* /usr/local/bin/
sudo chmod +x /usr/local/bin/instvisor-*
sudo cp deployments/systemd/instvisor.service /etc/systemd/system/
sudo systemctl daemon-reload
```

**B) Permission denied:**
```bash
# Make script executable
chmod +x scripts/install.sh
sudo ./scripts/install.sh
```

**C) Missing dependencies:**
```bash
# Install SQLite if needed
sudo apt-get install sqlite3 libsqlite3-dev  # Ubuntu/Debian
sudo yum install sqlite sqlite-devel         # RHEL/CentOS
```

---

### Issue: Docker container won't start

**Symptom:**
```bash
$ docker ps -a | grep instvisor
instvisor   Exited (1) 2 seconds ago
```

**Diagnosis:**
```bash
# Check container logs
docker logs instvisor

# Check container creation
docker inspect instvisor | grep -A 10 "State"
```

**Common causes and solutions:**

**A) Missing privileged mode:**
```bash
# Wrong
docker run abhishekkarki/instvisor

# Correct
docker run --privileged --pid=host \
  -v /:/rootfs:ro \
  -v /sys:/sys:ro \
  abhishekkarki/instvisor
```

**B) Volume mount error:**
```bash
# Check if volumes exist
docker volume ls | grep instvisor

# Recreate container with correct mounts
docker rm instvisor
docker run -d --name instvisor \
  --privileged --pid=host \
  -v /:/rootfs:ro \
  -v /sys:/sys:ro \
  -v instvisor-data:/var/lib/instvisor \
  abhishekkarki/instvisor:latest
```

**C) Port conflict (if exporter enabled):**
```bash
# Check if port 9090 is in use
sudo lsof -i :9090

# Stop conflicting service or change port
docker run -p 9091:9090 ...
```

---

## Service/Agent Issues

### Issue: Systemd service fails to start

**Symptom:**
```bash
$ sudo systemctl start instvisor
Job for instvisor.service failed because the control process exited with error code.
```

**Diagnosis:**
```bash
# Check detailed status
sudo systemctl status instvisor -l

# Check recent logs
sudo journalctl -u instvisor -n 50 --no-pager

# Check service file
sudo cat /etc/systemd/system/instvisor.service
```

**Common causes:**

**A) Binary missing:**
```bash
# Error: /usr/local/bin/instvisor-agent: No such file or directory

# Solution: Reinstall
cd ~/instvisor
sudo make install
```

**B) Config file missing:**
```bash
# Error: Failed to open config file: /etc/instvisor/agent.yaml

# Solution: Copy config
sudo mkdir -p /etc/instvisor
sudo cp configs/agent.yaml /etc/instvisor/
```

**C) Database directory not writable:**
```bash
# Error: Failed to open database: permission denied

# Solution: Create directory and set permissions
sudo mkdir -p /var/lib/instvisor
sudo chown instvisor:instvisor /var/lib/instvisor
sudo chmod 755 /var/lib/instvisor
```

**D) Systemd service file corrupted:**
```bash
# Reinstall service file
sudo cp deployments/systemd/instvisor.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl start instvisor
```

---

### Issue: Service starts but immediately crashes

**Symptom:**
```bash
$ sudo systemctl status instvisor
Active: failed (Result: exit-code)
```

**Diagnosis:**
```bash
# Check crash logs
sudo journalctl -u instvisor -n 100 | grep -i error

# Run agent manually to see error
sudo -u instvisor /usr/local/bin/instvisor-agent -config /etc/instvisor/agent.yaml
```

**Common causes:**

**A) Permission denied on /proc or /sys:**
```bash
# Error: permission denied: /proc/stat

# Solution: Run as root (instvisor needs root for /proc access)
# Edit service file:
sudo nano /etc/systemd/system/instvisor.service

# Change:
User=instvisor
# To:
User=root

sudo systemctl daemon-reload
sudo systemctl restart instvisor
```

**B) SQLite library missing:**
```bash
# Error: error while loading shared libraries: libsqlite3.so.0

# Solution: Install SQLite
sudo apt-get install libsqlite3-0  # Ubuntu/Debian
sudo yum install sqlite            # RHEL/CentOS
```

**C) Invalid config YAML:**
```bash
# Error: yaml: line X: mapping values are not allowed

# Solution: Validate YAML
python3 -c "import yaml; yaml.safe_load(open('/etc/instvisor/agent.yaml'))"

# Or use online validator: yamllint.com
```

---

## No Metrics Collected

### Issue: Agent running but no metrics in database

**Symptom:**
```bash
$ sudo sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"
0
```

**Diagnosis:**
```bash
# Check if agent is actually running
sudo systemctl status instvisor

# Check logs for collection messages
sudo journalctl -u instvisor -f

# Expected every 15 seconds:
# "Collected and stored X metrics from cpu"
# "Collected and stored X metrics from memory"
```

**Solutions:**

**A) Agent just started (need 2 collection cycles):**
```bash
# Wait 30 seconds (2 × 15s interval)
# Then check again
sleep 30
sudo sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"
```

**B) Collectors disabled in config:**
```bash
# Check config
sudo cat /etc/instvisor/agent.yaml | grep -A 2 "collectors:"

# Ensure collectors are enabled:
collectors:
  cpu:
    enabled: true  # Must be true
  memory:
    enabled: true
```

**C) Database path incorrect:**
```bash
# Check what database agent is using
sudo journalctl -u instvisor | grep -i database

# Verify path matches config
sudo cat /etc/instvisor/agent.yaml | grep path
```

**D) /proc or /sys not mounted:**
```bash
# Check if /proc is available
ls /proc/stat

# Check if /sys is available
ls /sys/fs/cgroup/

# If missing, this is a serious system issue
# Check with: mount | grep proc
```

---

### Issue: Some metrics collected, others missing

**Symptom:**
```bash
# Only CPU metrics, no memory
$ sudo sqlite3 /var/lib/instvisor/metrics.db \
  "SELECT DISTINCT metric_name FROM metrics;"
cpu.usage
cpu.user
# (memory metrics missing)
```

**Diagnosis:**
```bash
# Check logs for collector errors
sudo journalctl -u instvisor | grep -i "memory\|error"

# Check if specific collector is enabled
sudo cat /etc/instvisor/agent.yaml | grep -A 5 "memory:"
```

**Solutions:**

**A) Collector disabled:**
```bash
# Edit config
sudo nano /etc/instvisor/agent.yaml

# Enable collector:
collectors:
  memory:
    enabled: true  # Change from false

# Restart
sudo systemctl restart instvisor
```

**B) Collector failing silently:**
```bash
# Run agent in foreground to see errors
sudo systemctl stop instvisor
sudo /usr/local/bin/instvisor-agent -config /etc/instvisor/agent.yaml

# Watch for errors from specific collectors
```

**C) Permission issue on specific /proc files:**
```bash
# Check readable
cat /proc/meminfo
cat /proc/diskstats
cat /proc/net/dev

# If permission denied, run agent as root
```

---

### Issue: First collection has very few metrics

**Symptom:**
```bash
# First log entry
Collected and stored 3 metrics from cpu

# Second log entry (15s later)
Collected and stored 28 metrics from cpu
```

**Explanation:** This is **normal behavior**, not a bug!

**Why?** 
- CPU, disk, and network collectors calculate **rates** (deltas)
- First collection stores baseline values
- Second collection calculates: `(current - baseline) / time_interval`
- First collection: Only stores cumulative counters (3 metrics)
- Second collection: Stores rates + cumulative (28 metrics)

**No action needed** - this is expected.

---

## Container Metrics Not Working

### Issue: No container metrics despite Docker running

**Symptom:**
```bash
# Docker is running
$ docker ps
CONTAINER ID   IMAGE     NAMES
abc123         nginx     my-nginx

# But no container metrics
$ sudo sqlite3 /var/lib/instvisor/metrics.db \
  "SELECT COUNT(*) FROM metrics WHERE metric_name LIKE 'container.%';"
0
```

**Diagnosis:**
```bash
# Check if container collector is enabled
sudo cat /etc/instvisor/agent.yaml | grep -A 2 "container:"

# Check cgroup v2 is available
stat -fc %T /sys/fs/cgroup/
# Should output: cgroup2fs

# Check if Docker cgroups exist
ls /sys/fs/cgroup/system.slice/docker-*.scope

# Check logs for container collector errors
sudo journalctl -u instvisor | grep -i container
```

**Solutions:**

**A) Container collector disabled:**
```bash
# Edit config
sudo nano /etc/instvisor/agent.yaml

# Enable:
collectors:
  container:
    enabled: true

# Restart
sudo systemctl restart instvisor
```

**B) cgroup v1 instead of v2:**
```bash
# Check cgroup version
stat -fc %T /sys/fs/cgroup/

# If output is "tmpfs" (cgroup v1), need to migrate to v2
# Add to /etc/default/grub:
GRUB_CMDLINE_LINUX="systemd.unified_cgroup_hierarchy=1"

# Update grub and reboot
sudo update-grub
sudo reboot

# After reboot, verify v2
stat -fc %T /sys/fs/cgroup/
# Should now be: cgroup2fs
```

**C) Docker using different cgroup path:**
```bash
# Check where Docker puts cgroups
docker inspect <container-id> | grep -i cgroup

# If not in /sys/fs/cgroup/system.slice/, this is a Docker config issue
# Check: cat /etc/docker/daemon.json
```

**D) No containers running when collector started:**
```bash
# Instvisor finds containers at startup
# If containers start AFTER instvisor, restart instvisor:
sudo systemctl restart instvisor

# Or just wait - collector checks every interval
```

---

### Issue: Container names show as IDs

**Symptom:**
```bash
=== CONTAINER BREAKDOWN ===
  27995b46e30e: 15% CPU    # ID instead of name
  cf90cecaaee0: 10% CPU
```

**Diagnosis:**
```bash
# Check if Docker config files are readable
ls -la /var/lib/docker/containers/

# Try to read a config
sudo cat /var/lib/docker/containers/27995b46e30e*/config.v2.json
```

**Solutions:**

**A) Permission denied on /var/lib/docker:**
```bash
# Grant read access (configs are not sensitive)
sudo chmod o+rX /var/lib/docker/containers/

# For Docker installation, mount the volume:
docker run -v /var/lib/docker:/var/lib/docker:ro ...
```

**B) Docker config format changed:**
```bash
# Check if config exists
sudo find /var/lib/docker/containers/ -name "config.v2.json"

# If missing, Docker may use different format
# Instvisor will fall back to IDs (metrics still work)
```

**Workaround:** Metrics work fine, just displayed with IDs. Map IDs to names manually:
```bash
docker ps --format "table {{.ID}}\t{{.Names}}"
```

---

## Analysis Issues

### Issue: "No data found" when running analysis

**Symptom:**
```bash
$ sudo instvisor-analyze -days 7
Error: No data found for analysis period
```

**Diagnosis:**
```bash
# Check if database has any data
sudo sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"

# Check time range of data
sudo sqlite3 /var/lib/instvisor/metrics.db \
  "SELECT datetime(MIN(timestamp), 'unixepoch'), 
          datetime(MAX(timestamp), 'unixepoch') 
   FROM metrics;"

# Check if agent is collecting
sudo systemctl status instvisor
```

**Solutions:**

**A) Database is empty:**
```bash
# Agent not collecting - see "No Metrics Collected" section above
sudo journalctl -u instvisor -n 50
```

**B) Not enough data yet:**
```bash
# Check how long agent has been running
sudo systemctl status instvisor | grep "Active:"

# If less than requested period (7 days), reduce days:
sudo instvisor-analyze -days 1  # Requires at least 1 day of data
```

**C) Clock skew / wrong timezone:**
```bash
# Check system time
date
timedatectl

# Check database timestamps
sudo sqlite3 /var/lib/instvisor/metrics.db \
  "SELECT datetime(timestamp, 'unixepoch') FROM metrics LIMIT 5;"

# If timestamps are in the future, time is wrong
```

**D) Wrong database path:**
```bash
# Check where analyze is looking
instvisor-analyze -h | grep -A 1 "db"

# Specify correct path
sudo instvisor-analyze -db /var/lib/instvisor/metrics.db -days 1
```

---

### Issue: Analysis is very slow

**Symptom:**
```bash
$ sudo instvisor-analyze -days 90
# Takes 5+ minutes to complete
```

**Diagnosis:**
```bash
# Check database size
sudo du -h /var/lib/instvisor/metrics.db

# Count total metrics
sudo sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"

# Check if indexes exist
sudo sqlite3 /var/lib/instvisor/metrics.db ".schema metrics"
```

**Solutions:**

**A) Database too large (>10GB):**
```bash
# Reduce retention period
sudo nano /etc/instvisor/agent.yaml
# Change: retention_days: 30  (from 90)

# Restart to apply
sudo systemctl restart instvisor

# Manually delete old data
sudo sqlite3 /var/lib/instvisor/metrics.db \
  "DELETE FROM metrics WHERE timestamp < strftime('%s', 'now', '-30 days');"

# Vacuum to reclaim space
sudo sqlite3 /var/lib/instvisor/metrics.db "VACUUM;"
```

**B) Missing indexes:**
```bash
# Recreate indexes (if missing)
sudo sqlite3 /var/lib/instvisor/metrics.db <<EOF
CREATE INDEX IF NOT EXISTS idx_metric_time ON metrics(metric_name, timestamp);
CREATE INDEX IF NOT EXISTS idx_timestamp ON metrics(timestamp);
EOF
```

**C) Running on slow disk:**
```bash
# Check disk I/O during analysis
iostat -x 1

# If util is 100%, disk is bottleneck
# Move database to faster disk (SSD)
sudo systemctl stop instvisor
sudo mv /var/lib/instvisor/metrics.db /mnt/ssd/
sudo ln -s /mnt/ssd/metrics.db /var/lib/instvisor/metrics.db
sudo systemctl start instvisor
```

**D) Analyze shorter periods:**
```bash
# Instead of 90 days, analyze 7 or 30
sudo instvisor-analyze -days 7  # Much faster
```

---

### Issue: Recommendations seem wrong

**Symptom:**
```bash
Recommendation: 1 vCPU, 2 GB
# But I know my app needs more
```

**Diagnosis:**
```bash
# Check analysis period
sudo instvisor-analyze -days 7 | head -20

# Look at P95 values
# Check workload pattern

# Review actual usage
sudo sqlite3 /var/lib/instvisor/metrics.db \
  "SELECT AVG(value), MAX(value) 
   FROM metrics 
   WHERE metric_name = 'cpu.usage';"
```

**Common causes:**

**A) Collection period too short:**
```bash
# Only 1-2 days of data misses weekly patterns
# Solution: Wait for 7+ days, then re-analyze
```

**B) Analyzed during low-usage period:**
```bash
# Example: Analyzed weekend, but weekdays are busy
# Solution: Analyze longer period that includes peaks
sudo instvisor-analyze -days 14  # Include multiple weeks
```

**C) Recent deployment changed workload:**
```bash
# Old data (low usage) + new data (high usage) = mixed signal
# Solution: Wait 7 days after change, analyze fresh
```

**D) Max usage hitting 95%+ (already undersized):**
```bash
# Check Max value in analysis output
# If Max > 95%, you're hitting limits already
# Solution: Ignore downsize recommendation, investigate performance
```

---

## Performance Issues

### Issue: Instvisor using too much CPU

**Symptom:**
```bash
$ top
PID    COMMAND      %CPU
1234   instvisor-ag  25%   # Too high!
```

**Diagnosis:**
```bash
# Check collection interval
sudo cat /etc/instvisor/agent.yaml | grep interval

# Check number of collectors
sudo journalctl -u instvisor | grep "Starting collector"

# Check database size
sudo du -h /var/lib/instvisor/metrics.db
```

**Solutions:**

**A) Collection interval too short:**
```bash
# If interval is 5s or 10s, increase it
sudo nano /etc/instvisor/agent.yaml

# Change:
collection:
  interval: 30s  # Or even 1m

sudo systemctl restart instvisor
```

**B) Too many per-core metrics:**
```bash
# On systems with 64+ cores, disable per_core
sudo nano /etc/instvisor/agent.yaml

# Change:
collectors:
  cpu:
    per_core: false  # Only aggregate

sudo systemctl restart instvisor
```

**C) Database writes causing I/O:**
```bash
# Check I/O wait
iostat -x 1

# If high, reduce write frequency or move DB to tmpfs (loses data on reboot)
# NOT recommended, but for testing:
sudo systemctl stop instvisor
sudo mount -t tmpfs -o size=1G tmpfs /var/lib/instvisor
sudo systemctl start instvisor
```

**Expected CPU usage:** <1% on modern systems with default config (15s interval).

---

### Issue: Instvisor using too much memory

**Symptom:**
```bash
$ ps aux | grep instvisor
instvisor  1234  0.5  5.0  524288  ...   # 512 MB RAM
```

**Diagnosis:**
```bash
# Check if memory leak (growing over time)
# Monitor RSS over 24 hours:
while true; do ps aux | grep instvisor-agent | grep -v grep; sleep 3600; done

# Check database size
sudo du -h /var/lib/instvisor/metrics.db
```

**Solutions:**

**A) Normal behavior for large database:**
```bash
# SQLite caches data in memory
# If database is 5GB, some memory usage is expected
# Not a leak if stable
```

**B) Actual memory leak:**
```bash
# Restart agent
sudo systemctl restart instvisor

# If memory grows again, report bug:
# https://github.com/abhishekkarki/instvisor/issues
```

**Expected memory usage:** 20-100 MB depending on database size.

---

## Database Issues

### Issue: Database file is huge

**Symptom:**
```bash
$ sudo du -h /var/lib/instvisor/metrics.db
15G    /var/lib/instvisor/metrics.db
```

**Diagnosis:**
```bash
# Check retention setting
sudo cat /etc/instvisor/agent.yaml | grep retention

# Check actual data age
sudo sqlite3 /var/lib/instvisor/metrics.db \
  "SELECT datetime(MIN(timestamp), 'unixepoch'), 
          datetime(MAX(timestamp), 'unixepoch') 
   FROM metrics;"

# Count rows
sudo sqlite3 /var/lib/instvisor/metrics.db "SELECT COUNT(*) FROM metrics;"
```

**Solutions:**

**A) Reduce retention:**
```bash
# Edit config
sudo nano /etc/instvisor/agent.yaml

# Change:
collection:
  retention_days: 30  # From 90 or 365

sudo systemctl restart instvisor
```

**B) Manual cleanup:**
```bash
# Delete old data
sudo sqlite3 /var/lib/instvisor/metrics.db \
  "DELETE FROM metrics WHERE timestamp < strftime('%s', 'now', '-30 days');"

# Vacuum to reclaim space
sudo sqlite3 /var/lib/instvisor/metrics.db "VACUUM;"

# Check new size
sudo du -h /var/lib/instvisor/metrics.db
```

**C) Increase collection interval:**
```bash
# Collect less frequently
sudo nano /etc/instvisor/agent.yaml

# Change:
collection:
  interval: 30s  # From 15s

sudo systemctl restart instvisor
```

**D) Disable unnecessary collectors:**
```bash
# If you don't need disk or network metrics
collectors:
  disk:
    enabled: false
  network:
    enabled: false
```

---

### Issue: Database corruption

**Symptom:**
```bash
$ sudo instvisor-analyze -days 7
Error: database disk image is malformed
```

**Diagnosis:**
```bash
# Try to open database
sudo sqlite3 /var/lib/instvisor/metrics.db "PRAGMA integrity_check;"

# If corrupted, shows errors
```

**Solutions:**

**A) Attempt recovery:**
```bash
# Stop agent
sudo systemctl stop instvisor

# Backup corrupted DB
sudo cp /var/lib/instvisor/metrics.db /var/lib/instvisor/metrics.db.corrupted

# Try to dump and restore
sudo sqlite3 /var/lib/instvisor/metrics.db.corrupted .dump | \
  sudo sqlite3 /var/lib/instvisor/metrics.db.recovered

# If successful, replace
sudo mv /var/lib/instvisor/metrics.db.recovered /var/lib/instvisor/metrics.db

# Restart
sudo systemctl start instvisor
```

**B) Start fresh (lose historical data):**
```bash
# Stop agent
sudo systemctl stop instvisor

# Backup old DB
sudo mv /var/lib/instvisor/metrics.db /var/lib/instvisor/metrics.db.old

# Restart (will create new DB)
sudo systemctl start instvisor
```

**Prevention:**
- Don't kill instvisor with `SIGKILL` (use `systemctl stop`)
- Ensure disk is not full
- Use EXT4 or XFS filesystem (not FAT32)

---

### Issue: Database locked

**Symptom:**
```bash
$ sudo instvisor-analyze -days 7
Error: database is locked
```

**Cause:** SQLite allows only one writer at a time. If agent is writing, analyze must wait.

**Solutions:**

**A) Wait and retry:**
```bash
# Usually resolves in <1 second
sleep 2
sudo instvisor-analyze -days 7
```

**B) Stop agent during analysis (not recommended):**
```bash
sudo systemctl stop instvisor
sudo instvisor-analyze -days 7
sudo systemctl start instvisor
```

**C) If persistently locked:**
```bash
# Check if agent is stuck
sudo systemctl status instvisor

# Check for stale lock
sudo lsof /var/lib/instvisor/metrics.db

# If stale, restart agent
sudo systemctl restart instvisor
```

---

## Permission Issues

### Issue: Permission denied on /proc or /sys

**Symptom:**
```bash
Error: permission denied: /proc/stat
Error: permission denied: /sys/fs/cgroup/
```

**Cause:** Instvisor needs root access to read system files.

**Solutions:**

**A) Run as root (systemd):**
```bash
# Edit service file
sudo nano /etc/systemd/system/instvisor.service

# Change User line:
User=root

# Reload and restart
sudo systemctl daemon-reload
sudo systemctl restart instvisor
```

**B) Run as root (Docker):**
```bash
# Ensure --privileged flag
docker run --privileged --pid=host ...
```

**C) Grant capabilities (advanced):**
```bash
# Instead of full root, grant specific capabilities
sudo setcap cap_sys_admin,cap_dac_read_search+ep /usr/local/bin/instvisor-agent

# Update service to run as instvisor user
# (capabilities persist)
```

---

### Issue: Permission denied on database

**Symptom:**
```bash
Error: unable to open database file: permission denied
```

**Solutions:**

**A) Fix directory ownership:**
```bash
sudo chown -R instvisor:instvisor /var/lib/instvisor
sudo chmod 755 /var/lib/instvisor
```

**B) Fix database file:**
```bash
sudo chown instvisor:instvisor /var/lib/instvisor/metrics.db
sudo chmod 644 /var/lib/instvisor/metrics.db
```

**C) SELinux blocking (RHEL/CentOS):**
```bash
# Check SELinux status
getenforce

# If Enforcing, temporarily disable
sudo setenforce 0

# If that fixes it, create proper SELinux policy:
sudo ausearch -m avc -ts recent | audit2allow -M instvisor
sudo semodule -i instvisor.pp

# Re-enable SELinux
sudo setenforce 1
```

---

## Getting Help

### Gather Debug Information

Before reporting an issue, collect this information:
```bash
# 1. System info
uname -a
cat /etc/os-release

# 2. Instvisor version
instvisor-agent --version
# Or check Docker image tag
docker inspect instvisor | grep Image

# 3. Service status
sudo systemctl status instvisor -l

# 4. Recent logs
sudo journalctl -u instvisor -n 200 --no-pager > instvisor-logs.txt

# 5. Configuration
sudo cat /etc/instvisor/agent.yaml > instvisor-config.yaml

# 6. Database info
sudo sqlite3 /var/lib/instvisor/metrics.db <<EOF
.schema
SELECT COUNT(*) FROM metrics;
SELECT datetime(MIN(timestamp), 'unixepoch'), 
       datetime(MAX(timestamp), 'unixepoch') 
FROM metrics;
EOF

# 7. Disk space
df -h /var/lib/instvisor

# 8. cgroup version
stat -fc %T /sys/fs/cgroup/

# 9. Docker info (if relevant)
docker --version
docker ps -a | grep instvisor
```

### Where to Get Help

**1. Check Documentation**
- [Getting Started Guide](getting-started.md)
- [Configuration Reference](configuration.md)
- [Understanding Recommendations](recommendations.md)
- [Container Metrics Guide](container-metrics.md)

**2. Search Existing Issues**
- https://github.com/abhishekkarki/instvisor/issues

**3. GitHub Discussions (Questions)**
- https://github.com/abhishekkarki/instvisor/discussions

**4. Report a Bug**
- https://github.com/abhishekkarki/instvisor/issues/new
- Include debug information from above

**5. Community Support**
- Check GitHub Discussions for similar questions
- Tag issues with appropriate labels

---

## Common Error Messages Reference

Quick lookup for error messages:

| Error Message | Section | Quick Fix |
|---------------|---------|-----------|
| `command not found` | [Binary not found](#issue-binary-not-found-in-path) | Add to PATH or reinstall |
| `permission denied: /proc/stat` | [Permission Issues](#permission-issues) | Run as root |
| `database is locked` | [Database locked](#issue-database-locked) | Wait and retry |
| `no data found` | [Analysis Issues](#issue-no-data-found-when-running-analysis) | Wait longer or reduce days |
| `failed to create user` | [Installation fails](#issue-installation-script-fails) | User already exists (safe to ignore) |
| `cgroup2fs` not found | [Container metrics](#issue-no-container-metrics-despite-docker-running) | Enable cgroup v2 |
| `database disk image is malformed` | [Database corruption](#issue-database-corruption) | Attempt recovery or start fresh |
| `Job for instvisor.service failed` | [Service fails to start](#issue-systemd-service-fails-to-start) | Check logs with journalctl |

---

## Still Having Issues?

If this guide didn't solve your problem:

1. **Search GitHub Issues:** https://github.com/abhishekkarki/instvisor/issues
2. **Ask in Discussions:** https://github.com/abhishekkarki/instvisor/discussions
3. **Report a Bug:** Include all debug information listed in [Getting Help](#gather-debug-information)

**When reporting:**
- Include debug info (logs, config, system info)
- Describe what you tried from this guide
- Include error messages in full
- Mention your environment (OS, Docker version, etc.)

We're here to help!

---

## Related Documentation

- [Getting Started Guide](getting-started.md) - Installation and setup
- [Configuration Reference](configuration.md) - Config options
- [Understanding Recommendations](recommendations.md) - Analysis explained
- [Container Metrics Guide](container-metrics.md) - Container monitoring
- [GitHub Issues](https://github.com/abhishekkarki/instvisor/issues) - Known issues and bug reports