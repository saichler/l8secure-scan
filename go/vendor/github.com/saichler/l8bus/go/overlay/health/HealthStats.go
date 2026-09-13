package health

import (
	"github.com/saichler/l8types/go/ifs"
	"github.com/saichler/l8types/go/types/l8health"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

var cpuTracker = &CPUTracker{}

// CPUTracker tracks CPU usage for the current process by sampling /proc stats.
type CPUTracker struct {
	lastProcCPU uint64
	lastSysCPU  uint64
	lastSample  time.Time
	mu          sync.Mutex
}

func (c *CPUTracker) GetCPUUsage() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	// Only update CPU stats every 30 seconds to reduce syscall overhead
	if now.Sub(c.lastSample) < 30*time.Second && c.lastSample.Unix() > 0 {
		// Return cached calculation or 0 if no previous sample
		return 0
	}

	procStatData, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return 0
	}

	statFields := strings.Fields(string(procStatData))
	if len(statFields) < 17 {
		return 0
	}

	utime, _ := strconv.ParseUint(statFields[13], 10, 64)
	stime, _ := strconv.ParseUint(statFields[14], 10, 64)
	currentProcCPU := utime + stime

	systemStatData, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}

	systemStatLines := strings.Split(string(systemStatData), "\n")
	cpuLine := systemStatLines[0]
	cpuFields := strings.Fields(cpuLine)
	if len(cpuFields) < 8 {
		return 0
	}

	var currentSysCPU uint64
	for i := 1; i < len(cpuFields); i++ {
		val, _ := strconv.ParseUint(cpuFields[i], 10, 64)
		currentSysCPU += val
	}

	var cpuPercent float64
	// Calculate differential if we have previous values
	if c.lastSample.Unix() > 0 {
		procDelta := float64(currentProcCPU - c.lastProcCPU)
		sysDelta := float64(currentSysCPU - c.lastSysCPU)

		if sysDelta > 0 {
			cpuPercent = (procDelta / sysDelta) * 100.0
		}
	}

	// Update cached values
	c.lastProcCPU = currentProcCPU
	c.lastSysCPU = currentSysCPU
	c.lastSample = now

	return cpuPercent
}

func MemoryUsage() uint64 {
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)
	return memStats.Alloc
}

func BaseHealthStats(r ifs.IResources) *l8health.L8Health {
	stats := &l8health.L8HealthStats{}
	stats.MemoryUsage = MemoryUsage()
	stats.CpuUsage = cpuTracker.GetCPUUsage()

	hp := &l8health.L8Health{}
	hp.AUuid = r.SysConfig().LocalUuid
	hp.Status = l8health.L8HealthState_Up
	hp.Stats = stats
	hp.Services = r.Services().Services()

	return hp
}
