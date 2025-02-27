package hostdata

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// HostInfo holds system information.
type HostInfo struct {
	NodeID        string  `json:"node_id"`
	OSName        string  `json:"os_name"`
	OSVersion     string  `json:"os_version"`
	CPUCount      int     `json:"cpu_count"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemPercent    float64 `json:"mem_percent"`
	TotalMemory   uint64  `json:"total_memory"`
	NumContainers int     `json:"num_containers"`
}

// Dependency injection variables for easier testing.
var (
	hostnameFunc      = os.Hostname
	readFileFunc      = os.ReadFile
	cpuPercentFunc    = cpu.Percent
	virtualMemoryFunc = mem.VirtualMemory
)

// GetHostInfo returns system information using injected functions.
func GetHostInfo() (*HostInfo, error) {
	info := new(HostInfo)

	hostname, err := hostnameFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to get hostname: %w", err)
	}
	info.NodeID = hostname

	data, err := readFileFunc("/etc/os-release")
	if err != nil {
		return nil, fmt.Errorf("failed to read /etc/os-release: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "NAME=") {
			info.OSName = strings.Trim(line[len("NAME="):], `"`)
		} else if strings.HasPrefix(line, "VERSION=") {
			info.OSVersion = strings.Trim(line[len("VERSION="):], `"`)
		}
	}

	info.CPUCount = runtime.NumCPU()

	cpuPercents, err := cpuPercentFunc(250*time.Millisecond, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
	}
	info.CPUPercent = cpuPercents[0]

	vmStat, err := virtualMemoryFunc()
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual memory info: %w", err)
	}
	info.TotalMemory = vmStat.Total
	info.MemPercent = vmStat.UsedPercent

	return info, nil
}

// package hostdata

// import (
// 	"fmt"
// 	"os"
// 	"runtime"
// 	"strings"
// 	"time"

// 	"github.com/shirou/gopsutil/cpu"
// 	"github.com/shirou/gopsutil/mem"
// )

// type HostInfo struct {
// 	NodeID        string  "json:\"node_id\""
// 	OSName        string  "json:\"os_name\""
// 	OSVersion     string  "json:\"os_version\""
// 	CPUCount      int     "json:\"cpu_count\""
// 	CPUPercent    float64 "json:\"cpu_percent\""
// 	MemPercent    float64 "json:\"mem_percent\""
// 	TotalMemory   uint64  "json:\"total_memory\""
// 	NumContainers int     "json:\"num_containers\""
// }

// func GetHostInfo() (*HostInfo, error) {
// 	info := new(HostInfo)

// 	hostname, err := os.Hostname()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get hostname: %w", err)
// 	}
// 	info.NodeID = hostname

// 	lines := strings.Split(string(data), "\n")
// 	for _, line := range lines {
// 		if strings.HasPrefix(line, "NAME=") {
// 			// Remove potential quotes.
// 			info.OSName = strings.Trim(line[len("NAME="):], `"`)
// 		} else if strings.HasPrefix(line, "VERSION=") {
// 			info.OSVersion = strings.Trim(line[len("VERSION="):], `"`)
// 		}
// 	}

// 	// Get the total number of logical CPUs.
// 	info.CPUCount = runtime.NumCPU()

// 	cpuPercents, err := cpu.Percent(250*time.Millisecond, false)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get CPU usage: %w", err)
// 	}
// 	info.CPUPercent = cpuPercents[0]

// 	vmStat, err := mem.VirtualMemory()
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get virtual memory info: %w", err)
// 	}
// 	info.TotalMemory = vmStat.Total
// 	info.MemPercent = vmStat.UsedPercent

// 	return info, nil
// }
