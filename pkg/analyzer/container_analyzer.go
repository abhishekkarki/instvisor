package analyzer

import (
    "fmt"
    "sort"
    "time"

	"github.com/abhishekkarki/instvisor/pkg/metrics"
    "github.com/abhishekkarki/instvisor/pkg/storage"
)

// ContainerAnalyzer analyzes container-specific metrics
type ContainerAnalyzer struct {
    storage storage.Storage
}

// ContainerUsage represents resource usage for a single container
type ContainerUsage struct {
    ContainerID   string
    ContainerName string
    Image         string
    
    // CPU statistics
    CPUMean       float64
    CPUP50        float64
    CPUP90        float64
    CPUP95        float64
    CPUMax        float64
    
    // Memory statistics
    MemoryMean    float64
    MemoryP95     float64
    MemoryMaxGB   float64
    
    // Contribution to total host resources
    PercentOfHostCPU    float64
    PercentOfHostMemory float64
    
    Samples int
}

// ContainerBreakdown contains analysis of all containers
type ContainerBreakdown struct {
    TotalHostCPU    float64
    TotalHostMemory float64
    Containers      []ContainerUsage
    TopCPU          []ContainerUsage // Top 5 by CPU
    TopMemory       []ContainerUsage // Top 5 by Memory
    Period          time.Duration
}

// NewContainerAnalyzer creates a new container analyzer
func NewContainerAnalyzer(store storage.Storage) *ContainerAnalyzer {
    return &ContainerAnalyzer{
        storage: store,
    }
}

// AnalyzeContainers analyzes container resource usage over a period
func (ca *ContainerAnalyzer) AnalyzeContainers(period time.Duration, hostCPUP95, hostMemoryP95 float64) (*ContainerBreakdown, error) {
    end := time.Now()
    start := end.Add(-period)
    
    breakdown := &ContainerBreakdown{
        Period:          period,
        TotalHostCPU:    hostCPUP95,
        TotalHostMemory: hostMemoryP95,
        Containers:      make([]ContainerUsage, 0),
    }
    
    // Get list of unique containers from metrics
    containers, err := ca.getUniqueContainers(start, end)
    if err != nil {
        return nil, fmt.Errorf("failed to get containers: %w", err)
    }
    
    if len(containers) == 0 {
        // No containers, return empty breakdown
        return breakdown, nil
    }
    
    // Analyze each container
    for _, containerInfo := range containers {
        usage, err := ca.analyzeContainer(containerInfo, start, end)
        if err != nil {
            // Skip containers with errors, don't fail entire analysis
            continue
        }
        
        // Calculate percentage of host resources
        if hostCPUP95 > 0 {
            usage.PercentOfHostCPU = (usage.CPUP95 / hostCPUP95) * 100
        }
        if hostMemoryP95 > 0 {
            usage.PercentOfHostMemory = (usage.MemoryP95 / hostMemoryP95) * 100
        }
        
        breakdown.Containers = append(breakdown.Containers, *usage)
    }
    
    // Sort and get top consumers
    breakdown.TopCPU = ca.getTopN(breakdown.Containers, "cpu", 5)
    breakdown.TopMemory = ca.getTopN(breakdown.Containers, "memory", 5)
    
    return breakdown, nil
}

// getUniqueContainers gets list of unique containers from metrics
func (ca *ContainerAnalyzer) getUniqueContainers(start, end time.Time) ([]containerInfo, error) {
    // Query container CPU metrics to find all containers
    metrics, err := ca.storage.QueryMetrics("container.cpu.usage_percent", start, end, nil)
    if err != nil {
        return nil, err
    }
    
    // Extract unique container info from labels
    seen := make(map[string]containerInfo)
    for _, m := range metrics {
        containerID, ok := m.Labels["container_id"]
        if !ok {
            continue
        }
        
        if _, exists := seen[containerID]; !exists {
            seen[containerID] = containerInfo{
                ID:    containerID,
                Name:  m.Labels["container_name"],
                Image: m.Labels["image"],
            }
        }
    }
    
    // Convert map to slice
    containers := make([]containerInfo, 0, len(seen))
    for _, info := range seen {
        containers = append(containers, info)
    }
    
    return containers, nil
}

// analyzeContainer analyzes a single container
func (ca *ContainerAnalyzer) analyzeContainer(info containerInfo, start, end time.Time) (*ContainerUsage, error) {
    usage := &ContainerUsage{
        ContainerID:   info.ID,
        ContainerName: info.Name,
        Image:         info.Image,
    }
    
    // Query CPU metrics for this specific container
    cpuLabels := map[string]string{
        "container_id": info.ID,
    }
    
    cpuMetrics, err := ca.storage.QueryMetrics("container.cpu.usage_percent", start, end, cpuLabels)
    if err != nil {
        return nil, err
    }
    
    if len(cpuMetrics) > 0 {
        cpuValues := extractValues(cpuMetrics)
        usage.CPUMean = mean(cpuValues)
        usage.CPUP50 = percentile(cpuValues, 50)
        usage.CPUP90 = percentile(cpuValues, 90)
        usage.CPUP95 = percentile(cpuValues, 95)
        usage.CPUMax = max(cpuValues)
        usage.Samples = len(cpuValues)
    }
    
    // Query memory metrics for this container
    memMetrics, err := ca.storage.QueryMetrics("container.memory.usage_bytes", start, end, cpuLabels)
    if err == nil && len(memMetrics) > 0 {
        memValues := extractValues(memMetrics)
        usage.MemoryMean = mean(memValues) / 1024 / 1024 / 1024    // Convert to GB
        usage.MemoryP95 = percentile(memValues, 95) / 1024 / 1024 / 1024
        usage.MemoryMaxGB = max(memValues) / 1024 / 1024 / 1024
    }
    
    return usage, nil
}

// getTopN returns top N containers by resource usage
func (ca *ContainerAnalyzer) getTopN(containers []ContainerUsage, by string, n int) []ContainerUsage {
    if len(containers) == 0 {
        return []ContainerUsage{}
    }
    
    // Create a copy to avoid modifying original
    sorted := make([]ContainerUsage, len(containers))
    copy(sorted, containers)
    
    // Sort by specified metric
    sort.Slice(sorted, func(i, j int) bool {
        if by == "cpu" {
            return sorted[i].CPUP95 > sorted[j].CPUP95
        }
        return sorted[i].MemoryP95 > sorted[j].MemoryP95
    })
    
    // Return top N
    if len(sorted) < n {
        return sorted
    }
    return sorted[:n]
}

// Helper type for container info
type containerInfo struct {
    ID    string
    Name  string
    Image string
}

// Helper functions
func extractValues(metrics []metrics.Metric) []float64 {
    values := make([]float64, len(metrics))
    for i, m := range metrics {
        values[i] = m.Value
    }
    sort.Float64s(values)
    return values
}

func max(values []float64) float64 {
    if len(values) == 0 {
        return 0
    }
    maxVal := values[0]
    for _, v := range values {
        if v > maxVal {
            maxVal = v
        }
    }
    return maxVal
}