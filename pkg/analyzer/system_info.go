package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// SystemInfo contains information about the current system
type SystemInfo struct {
	CPUCores int
	MemoryGB float64
	Hostname string
	OS       string
	Arch     string
}

// DetectSystemInfo detects current system configuration
func DetectSystemInfo() (*SystemInfo, error) {
	info := &SystemInfo{
		CPUCores: runtime.NumCPU(),
		Arch:     runtime.GOARCH,
		OS:       runtime.GOOS,
	}

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil {
		info.Hostname = hostname
	}

	// Get total memory from /proc/meminfo
	memGB, err := getTotalMemoryGB()
	if err == nil {
		info.MemoryGB = memGB
	}

	return info, nil
}

func getTotalMemoryGB() (float64, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				kb, err := strconv.ParseUint(fields[1], 10, 64)
				if err != nil {
					return 0, err
				}
				// Convert KB to GB
				return float64(kb) / 1024.0 / 1024.0, nil
			}
		}
	}

	return 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
}

func (s *SystemInfo) String() string {
	return fmt.Sprintf("%s - %d vCPUs, %.1f GB RAM (%s/%s)",
		s.Hostname, s.CPUCores, s.MemoryGB, s.OS, s.Arch)
}
