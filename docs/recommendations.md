# Understanding Recommendations

This guide explains how instvisor analyzes your system and generates right-sizing recommendations.

---

## Overview

Instvisor's recommendation engine follows this process:
```
Collect Metrics → Calculate Statistics → Detect Pattern → Apply Headroom → Match Instances → Recommend
```

Each step is explained in detail below.

---

## How Recommendations Work

### Step 1: Data Collection

Instvisor collects metrics every 15 seconds (configurable) over your chosen time period:
```
7 days × 24 hours × 60 minutes × 4 samples/min = 40,320 CPU samples
```

For each metric (CPU, memory, disk, network), we store:
- Timestamp
- Value
- Labels (which CPU, which container, etc.)

### Step 2: Statistical Analysis

From these samples, instvisor calculates:

| Statistic | Description | Use Case |
|-----------|-------------|----------|
| **Min** | Lowest value seen | Baseline usage |
| **Mean** | Average value | Typical load |
| **P50 (Median)** | 50% of samples below this | Middle point |
| **P90** | 90% of samples below this | Common peak |
| **P95** | 95% of samples below this | **Used for sizing** ⭐ |
| **P99** | 99% of samples below this | Rare peaks |
| **Max** | Highest value seen | Absolute peak |
| **StdDev** | Variation from mean | Workload stability |

**Example CPU data:**
```
Min:    2.1%
Mean:   25.3%   ← Average
P50:    24.8%
P90:    42.1%
P95:    52.3%   ← This drives the recommendation
P99:    68.7%
Max:    95.2%
StdDev: 15.2    ← Used for pattern detection
```

---

## Why P95 (Not Average, Not Max)?

This is the most important question to understand.

### The Problem with Other Metrics

#### **Using Average (Mean)**
```
Your average CPU: 25%
Sized for: 25% × 1.2 headroom = 30%

Result: You'll run out of CPU during normal peaks (which happen 50% of the time!)
```

**Example scenario:**
- Mean: 25% CPU
- P95: 52% CPU
- If sized for mean: 40% of the time you're CPU-starved
- Users experience slowness during normal business hours

#### **Using P99 or Max**
```
Your P99 CPU: 85%
Sized for: 85% × 1.2 = 102% → Need more cores

But this spike happens <1% of the time (7 hours per month)
```

**Example scenario:**
- P95: 52% CPU
- P99: 85% CPU (monthly report generation)
- If sized for P99: You're paying for capacity used 1% of the time
- Waste: 40% over-provisioned 99% of the month

### **Why P95 is the Sweet Spot**
```
P95 = 52% CPU
Sized for: 52% × 1.2 = 62%

Meaning:
- 95% of the time: You have enough resources 
- 5% of the time: Brief resource constraints (acceptable) 
- Cost: Optimal - not over-provisioning for rare events 
```

**Visual representation:**
```
Time spent at each CPU level (7 days):

0-30%:  ████████████████ 40% of time
30-50%: ███████████      30% of time
50-60%: ████             15% of time  } P95 covers this
60-70%: ██                8% of time  }
70-85%: █                 5% of time  ← Occasional spikes (acceptable)
85%+:   ▌                 2% of time  ← Rare (P99)
```

**Key insight:** Sizing for P95 means you're comfortable with brief resource constraints 5% of the time. This is acceptable because:

1. **Modern systems handle brief overload well** (OS scheduling, request queuing)
2. **5% is ~84 minutes per day** - spread across the day, not continuous
3. **Cost savings of 30-50%** justify occasional slowness
4. **You can add headroom** if you need higher tolerance

---

## Headroom Calculation

After determining P95 usage, instvisor adds safety margin (headroom).

### Default Headroom

- **CPU**: 20%
- **Memory**: 15%

### Why Different Percentages?

**CPU (20% headroom):**
- CPU is **elastic** - can burst temporarily
- OS scheduler handles brief overload gracefully
- 20% buffer allows for growth and variability

**Memory (15% headroom):**
- Memory is **inelastic** - running out = OOM kills
- Memory usage is more predictable than CPU
- 15% buffer prevents swapping and OOM

### Example Calculation
```
CPU Analysis:
  P95: 42%
  Headroom: 20%
  Required: 42% × 1.20 = 50.4%

If current: 8 vCPUs
  Current capacity: 8 cores = 100%
  Required capacity: 50.4%
  Recommended cores: 8 × 0.504 = 4.03 → 4 vCPUs

Memory Analysis:
  P95: 12 GB (out of 32 GB = 37.5%)
  Headroom: 15%
  Required: 12 GB × 1.15 = 13.8 GB → 14 GB

Recommendation: 4 vCPUs, 14 GB RAM
Savings: 50% CPU, 56% memory
```

### When to Adjust Headroom

**Increase headroom (25-30%) when:**
- Mission-critical application (zero tolerance for slowness)
- Rapidly growing workload
- Unpredictable traffic spikes
- Compliance/SLA requirements

**Decrease headroom (10-15%) when:**
- Cost optimization is priority
- Workload is very stable (low StdDev)
- Easy to scale up if needed (cloud auto-scaling)
- Non-production environments

**How to adjust:**
```bash
instvisor-analyze -days 7 -headroom-cpu 30 -headroom-mem 20
```

---

## Workload Pattern Detection

Instvisor analyzes your workload stability and categorizes it into three patterns.

### Pattern: Steady-State

**Characteristics:**
- Low variation (StdDev < 20% of mean)
- Range ratio < 1.5 (Max / Mean)
- Predictable, constant load

**Example:**
```
CPU Usage over 7 days:
Mean:   25%
StdDev: 4%     ← Very low variation
Max:    32%
CV:     0.16   ← Coefficient of Variation (StdDev/Mean)

Pattern: steady_state (confidence: 90%)
```

**Recommendation impact:**
- Can size closer to P95 (high confidence)
- Standard instances with reserved capacity
- Cost-effective to use reserved/committed instances

**Real-world examples:**
- Web servers with consistent traffic
- Database with stable query load
- API services with predictable usage

---

### Pattern: Bursty

**Characteristics:**
- High variation (StdDev > 50% of mean)
- Large range ratio > 3.0
- Unpredictable spikes

**Example:**
```
CPU Usage over 7 days:
Mean:   25%
StdDev: 18%    ← High variation
Max:    95%    ← Sudden spikes
CV:     0.72

Pattern: bursty (confidence: 80%)
```

**Recommendation impact:**
- May need higher headroom (25-30%)
- Consider burstable instances (AWS t3, Azure B-series)
- Or use auto-scaling instead of static sizing

**Real-world examples:**
- E-commerce during flash sales
- Social media during viral events
- Video encoding workloads
- CI/CD build servers

**Special recommendation for bursty workloads:**
```
Alternative to larger instance:
1. Use burstable instances (AWS t3.large vs m5.large)
2. Enable auto-scaling (scale from 2→4 instances during bursts)
3. Add caching layer (reduce backend load during spikes)
```

---

### Pattern: Scheduled

**Characteristics:**
- Moderate variation (20% < StdDev < 50% of mean)
- Regular patterns (daily/weekly cycles)
- Predictable timing

**Example:**
```
CPU Usage over 7 days:
Mean:   30%
StdDev: 12%    ← Moderate variation
Max:    65%

Pattern: scheduled (confidence: 70%)

Usage pattern:
Mon-Fri 9am-5pm:  High (60%)
Nights/weekends:  Low (10%)
```

**Recommendation impact:**
- Consider scheduled auto-scaling
- Or use spot/preemptible instances for batch jobs
- Standard sizing for base load + scale for peaks

**Real-world examples:**
- Batch processing (nightly ETL jobs)
- Business hours applications
- Report generation (monthly/weekly)
- Backup workloads

---

## Instance Type Matching

After calculating required resources, instvisor maps to real cloud instance types.

### Instance Family Selection

Instvisor analyzes your **memory-to-CPU ratio** to recommend the right family:
```
Memory per vCPU ratio = Memory (GB) / vCPUs

Ratio < 2:    Compute-optimized (C-family)
Ratio 2-6:    General purpose (T, M-family)
Ratio > 6:    Memory-optimized (R, X-family)
```

### AWS Instance Type Mapping

| Family | Description | Ratio | Use Case |
|--------|-------------|-------|----------|
| **t3/t4g** | Burstable | 0.5-4 | Bursty workloads, development |
| **c5/c6** | Compute-optimized | 2 | CPU-intensive, batch processing |
| **m5/m6** | General purpose | 4 | Balanced workloads, web servers |
| **r5/r6** | Memory-optimized | 8 | Databases, caching, in-memory |

**Example matching:**
```
Required: 4 vCPUs, 16 GB RAM
Ratio: 16 / 4 = 4 GB/vCPU

Matches:
m5.xlarge   (4 vCPU, 16 GB) - Perfect fit
t3.xlarge   (4 vCPU, 16 GB) - If bursty workload
c5.xlarge   (4 vCPU, 8 GB)  - Not enough memory
r5.large    (2 vCPU, 16 GB) - Not enough CPU
```

### OTC (Open Telekom Cloud) Instance Mapping

| Family | Description | Ratio | Use Case |
|--------|-------------|-------|----------|
| **s3** | General purpose | 4 | Standard workloads |
| **c3** | Compute-optimized | 2 | CPU-intensive |
| **m3** | Memory-optimized | 8 | Memory-heavy |

**Naming format:** `{family}.{size}.{ratio}`
- Example: `s3.xlarge.4` = S3 family, xlarge size, 4GB RAM per vCPU

### Azure Instance Mapping

| Series | Description | Use Case |
|--------|-------------|----------|
| **B** | Burstable | Dev/test, low-usage |
| **D** | General purpose | Standard workloads |
| **F** | Compute-optimized | CPU-bound apps |
| **E** | Memory-optimized | Large databases |

---

## Container-Aware Insights

When containers are detected, instvisor provides additional context.

### Container Breakdown
```
Host CPU P95: 70%

Container breakdown:
  postgres:  49% CPU  (70% of host) ← Drives sizing
  nginx:      7% CPU  (10% of host)
  app:        5% CPU  (7% of host)
```

### Insight Triggers

Instvisor generates insights based on thresholds:

#### High Consumer (>60% of host)
```
Container 'postgres' consumes 70% of your host CPU
   Consider optimizing 'postgres' before scaling the entire host

Actions to try:
1. Run EXPLAIN ANALYZE on slow queries
2. Add database indexes
3. Increase postgres connection pooling
4. Consider read replicas
5. Move to managed RDS/CloudSQL
```

**Why this matters:** Scaling the host to 16 vCPUs won't help if postgres can't use >4 cores efficiently.

#### Moderate Consumer (40-60% of host)
```
Container 'postgres' is your largest CPU consumer (55% of host)
```

#### Distributed Load (<30% per container)
```
CPU usage is well-distributed. Top consumer: 'nginx' (25% of host)
Resource usage is evenly distributed across containers - no single bottleneck
```

**Insight:** This is ideal - no single container drives sizing, right-sizing the host helps all containers.

---

## Interpreting Recommendations

### Example 1: Simple Downsize
```
=== RECOMMENDATION ===
Current: 8 vCPUs, 32 GB
Recommended: 4 vCPUs, 16 GB
Savings: 50%

Rationale:
  • CPU P95: 42% with 20% headroom = 50%
  • Memory P95: 37% with 15% headroom = 43%
  • Steady workload - predictable usage
```

**How to act:**
1. **Trust this recommendation** - steady workload, clear savings
2. Test in staging first with 4 vCPUs
3. Monitor for 1-2 weeks in staging
4. If stable, apply to production during maintenance window

**Risk level:** Low - steady pattern, clear under-utilization

---

### Example 2: Bursty Workload Caveat
```
=== RECOMMENDATION ===
Current: 8 vCPUs, 32 GB
Recommended: 6 vCPUs, 24 GB
Savings: 25%

Rationale:
  • CPU P95: 68% with 20% headroom = 82%
  • Bursty workload - consider auto-scaling
```

**How to act:**
1. **Don't blindly downsize** - bursty = unpredictable
2. Options:
   - Keep 8 vCPUs (safer)
   - Downsize to 6 vCPUs + enable auto-scaling (scale 6→8 during bursts)
   - Use burstable instances (t3.2xlarge) that can burst above baseline
3. Monitor burst frequency and duration
4. Set up alerts for CPU >80% sustained for >5 minutes

**Risk level:** Medium - requires additional monitoring/auto-scaling

---

### Example 3: Container-Driven Sizing
```
=== RECOMMENDATION ===
Current: 8 vCPUs, 32 GB
Recommended: 6 vCPUs, 24 GB

=== CONTAINER INSIGHTS ===
Container 'postgres' consumes 70% of your host CPU
   Consider optimizing 'postgres' before scaling the host
```

**How to act:**
1. **Don't downsize yet** - investigate postgres first
2. Actions before sizing:
   - Check slow query log: `SELECT * FROM pg_stat_statements ORDER BY total_time DESC LIMIT 10;`
   - Add missing indexes
   - Tune postgres config (work_mem, shared_buffers)
   - Enable query caching
3. Re-analyze after optimization - might get to 4 vCPUs instead of 6
4. Alternative: Move postgres to managed service (RDS) sized independently

**Risk level:** High if you just downsize - optimize first!

---

### Example 4: Already Well-Sized
```
=== RECOMMENDATION ===
Current: 4 vCPUs, 16 GB
Recommended: 4 vCPUs, 16 GB
Savings: 0%

Rationale:
  • CPU P95: 72% with 20% headroom = 86%
  • Memory P95: 68% with 15% headroom = 78%
  • Current sizing is optimal
```

**How to act:**
1. **No action needed** - you're already right-sized
2. Monitor for growth - if P95 trends upward, plan to scale
3. Re-analyze monthly to catch gradual increases

**Risk level:** None - maintain current size

---

## When to Trust (or Question) Recommendations

### Trust the Recommendation When:

- Collection period ≥ 7 days
- Workload pattern is "steady_state"
- P95 and Mean are close (low variation)
- No recent infrastructure changes
- Container breakdown shows distributed load

### Question the Recommendation When:

- Collection period < 7 days (not enough data)
- Recent spike skewed P95 (check: is Max >> P95?)
- Major event during collection (Black Friday, product launch)
- One container dominates (>70% of host)
- Seasonal business (collected during slow period)

### Don't Follow the Recommendation If:

- Max > 95% (you're hitting limits already)
- You're in rapid growth phase (3-6 months old startup)
- Known upcoming traffic increase (planned marketing campaign)
- SLA requires <1% downtime (use P99 instead)

---

## Red Flags in Analysis

### Red Flag 1: Max ≈ 100%
```
CPU Usage:
  P95: 72%
  Max: 98%  ← WARNING!
```

**Meaning:** You're hitting resource limits, likely experiencing performance issues already.

**Action:** Do NOT downsize. Consider upsizing or optimization first.

---

### Red Flag 2: Sudden Spike in Data
```
CPU P95 over time:
Week 1-3: 40-45%
Week 4:   85%     ← What happened?
```

**Meaning:** Something changed recently (deployment, traffic spike, data import).

**Action:** Investigate week 4 spike before trusting overall P95. May need longer collection period.

---

### Red Flag 3: Extremely Low Usage
```
CPU Usage:
  Mean: 3%
  P95:  5%
  Max:  8%
```

**Meaning:** This host is doing almost nothing. Why does it exist?

**Action:** 
- Can this workload be consolidated to another host?
- Is this a dev/test environment that can be powered off when not in use?
- Consider serverless/container alternatives

---

## Advanced: Multi-Period Analysis

For complex workloads, analyze multiple periods:
```bash
# Last 7 days (recent)
instvisor-analyze -days 7

# Last 30 days (includes monthly patterns)
instvisor-analyze -days 30

# Last 90 days (seasonal trends)
instvisor-analyze -days 90
```

**Compare results:**
```
7-day P95:  42% CPU
30-day P95: 48% CPU  ← Monthly spike
90-day P95: 45% CPU

Recommendation: Size for 48% (30-day), not 42% (7-day)
Reason: Monthly report generation causes spike
```

---

## Cost Estimation (Future Feature)

Coming soon: Actual cost calculations based on cloud provider pricing.
```
Current cost:  €245/month (OTC s3.2xlarge.4)
Recommended:   €140/month (OTC s3.xlarge.4)
Monthly savings: €105 (43%)
Annual savings:  €1,260
```

Refer to issues.

---

## Best Practices

### 1. Always Start with 7+ Days of Data

Anything less than 7 days risks missing weekly patterns (weekend vs weekday traffic).

### 2. Re-analyze After Major Changes

After deployments, configuration changes, or marketing campaigns, wait 7 days and re-analyze.

### 3. Test Recommendations in Staging First

Never apply recommendations directly to production. Test with:
- Staging environment
- Canary deployment (10% of traffic)
- Blue/green deployment

### 4. Monitor After Downsizing

Set up alerts:
- CPU > 85% for 15+ minutes
- Memory > 90% for 5+ minutes
- Disk I/O wait > 20%

### 5. Consider Business Context

Technical metrics don't know about:
- Upcoming product launches
- End-of-quarter traffic
- Holiday season traffic
- Planned marketing campaigns

Adjust recommendations based on business knowledge.

---

## FAQ

### Q: Why not just use cloud provider's recommendation tools?

**A:** Cloud providers recommend based on short periods (14 days max) and don't provide container breakdown. Instvisor:
- Analyzes up to 365 days
- Shows which containers drive sizing
- Works across multiple clouds
- You control the data (not sent to cloud provider)

### Q: Can I use P99 instead of P95?

**A:** Yes:
```bash
# Use P99 for sizing (more conservative)
instvisor-analyze -days 7 -use-p99
```

Good for mission-critical workloads, but expect 20-30% higher costs.

### Q: What if my workload is growing?

**A:** Instvisor shows current state, not future growth. For growing workloads:
- Add growth buffer: If growing 10%/month, add 20-30% to recommendation
- Re-analyze monthly to track growth trend
- Use auto-scaling instead of static sizing

### Q: Should I size for peak traffic?

**A:** Usually no. Size for P95, then use:
- Auto-scaling for predictable peaks (known Black Friday traffic)
- Burstable instances for occasional spikes
- CDN/caching to reduce backend load during peaks

Exception: If peak traffic is business-critical (payment processing during sales), size for P99.

---

## Real-World Examples

### Example 1: E-commerce Website
```
Analysis (30 days including Black Friday):
  P95 CPU: 65%
  Max CPU: 98% (Black Friday spike)
  Pattern: Scheduled (high on weekends)

Recommendation: 6 vCPUs
Alternative recommendation: 4 vCPUs base + auto-scale to 8 vCPUs

Decision: Use auto-scaling
  - Base: 4 vCPUs (handles 90% of time)
  - Scale to 8 vCPUs when CPU > 70%
  - Cost: Pay for 4 vCPUs most of time, 8 during peaks
```

### Example 2: Database Server
```
Analysis (90 days):
  P95 CPU: 45%
  P95 Memory: 85%
  Container: postgres using 90% of memory

Recommendation: Don't change vCPUs, increase memory
  From: 4 vCPUs, 16 GB → 4 vCPUs, 24 GB
  
Action taken: 
  1. Tuned postgres (reduced connections)
  2. P95 memory dropped to 70%
  3. Kept 4 vCPUs, 16 GB (no resize needed!)
```

### Example 3: Microservices Platform
```
Analysis (14 days):
  Host P95 CPU: 70%
  Container breakdown:
    - api-gateway: 35% (50% of host)
    - auth-service: 14% (20% of host)
    - 10 other services: 21% (30% of host)

Recommendation: Optimize api-gateway
  - Added caching layer
  - Enabled HTTP/2
  - P95 dropped from 35% to 18%
  - Host P95 now: 45%
  - Downsized: 8 vCPUs → 4 vCPUs (50% cost savings)
```

---

## Summary

**Key Takeaways:**

1. **P95 is the sweet spot** - balances cost and performance
2. **Workload pattern matters** - steady vs bursty affects confidence
3. **Container insights are gold** - optimize heavy containers before scaling
4. **Context is critical** - technical metrics + business knowledge
5. **Test before applying** - staging, canary, then production
6. **Monitor after changes** - ensure recommendations work in practice

**Decision framework:**
```
Is collection ≥ 7 days? → No → Collect more data
                       ↓ Yes
Is pattern steady?     → No → Consider auto-scaling or higher headroom
                       ↓ Yes
Is any container >60%? → Yes → Optimize that container first
                       ↓ No
Is Max < 95%?          → No → Investigate performance issues first
                       ↓ Yes
Trust the recommendation!
```

---

## Related Documentation

- [Getting Started Guide](getting-started.md) - Installation and setup
- [Configuration Reference](configuration.md) - Adjust collection settings
- [Container Metrics Guide](container-metrics.md) - Understanding container data
- [Troubleshooting](troubleshooting.md) - Common issues

---

**Questions?** Open an issue on [GitHub](https://github.com/abhishekkarki/instvisor/issues) or discuss in [Discussions](https://github.com/abhishekkarki/instvisor/discussions).