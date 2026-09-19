//go:build linux

package metrics

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func NewLinuxCollector() (Collector, error) { return &linuxCollector{}, nil }

type linuxCollector struct{ previousTotal, previousIdle uint64 }

func (c *linuxCollector) Collect() (Status, error) {
	total, idle, err := readCPU()
	if err != nil {
		return Status{}, err
	}
	cpu := 0.0
	if c.previousTotal > 0 && total > c.previousTotal {
		deltaTotal, deltaIdle := total-c.previousTotal, idle-c.previousIdle
		cpu = 100 * float64(deltaTotal-deltaIdle) / float64(deltaTotal)
	}
	c.previousTotal, c.previousIdle = total, idle
	used, memoryTotal, err := readMemory()
	if err != nil {
		return Status{}, err
	}
	disk, err := readDiskPercent()
	if err != nil {
		return Status{}, err
	}
	load, err := readLoad1()
	if err != nil {
		return Status{}, err
	}
	uptime, err := readUptime()
	if err != nil {
		return Status{}, err
	}
	return Status{Timestamp: time.Now().UTC(), CPUPercent: cpu, MemoryUsedMiB: used, MemoryTotalMiB: memoryTotal, DiskUsedPercent: disk, Load1: load, TemperatureC: readTemperature(), UptimeSeconds: uptime}, nil
}

func readCPU() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, fmt.Errorf("read /proc/stat: %w", err)
	}
	fields := strings.Fields(strings.SplitN(string(data), "\n", 2)[0])
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, fmt.Errorf("invalid /proc/stat")
	}
	var total uint64
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, 0, err
		}
		total += value
	}
	idle, err := strconv.ParseUint(fields[4], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	if len(fields) > 5 {
		iowait, err := strconv.ParseUint(fields[5], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		idle += iowait
	}
	return total, idle, nil
}

func readMemory() (uint64, uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, fmt.Errorf("read /proc/meminfo: %w", err)
	}
	values := map[string]uint64{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err == nil {
			values[strings.TrimSuffix(fields[0], ":")] = value
		}
	}
	total, okTotal := values["MemTotal"]
	available, okAvailable := values["MemAvailable"]
	if !okTotal || !okAvailable || total == 0 || available > total {
		return 0, 0, fmt.Errorf("invalid /proc/meminfo")
	}
	return (total - available) / 1024, total / 1024, nil
}

func readDiskPercent() (float64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, err
	}
	if stat.Blocks == 0 {
		return 0, fmt.Errorf("root filesystem has no blocks")
	}
	return 100 * float64(stat.Blocks-stat.Bavail) / float64(stat.Blocks), nil
}
func readLoad1() (float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, fmt.Errorf("invalid /proc/loadavg")
	}
	return strconv.ParseFloat(fields[0], 64)
}
func readUptime() (uint64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, fmt.Errorf("invalid /proc/uptime")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	return uint64(seconds), err
}
func readTemperature() *float64 {
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return nil
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(data)), 64)
	if err != nil {
		return nil
	}
	value /= 1000
	return &value
}
