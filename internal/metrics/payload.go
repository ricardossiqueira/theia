// Package metrics collects host metrics and builds canonical telemetry payloads.
package metrics

import (
	"encoding/json"
	"fmt"
	"time"
)

type Status struct {
	Timestamp       time.Time
	CPUPercent      float64
	MemoryUsedMiB   uint64
	MemoryTotalMiB  uint64
	DiskUsedPercent float64
	Load1           float64
	TemperatureC    *float64
	UptimeSeconds   uint64
}

type Collector interface {
	Collect() (Status, error)
}

// StatusFields encodes the domain fields of status, matching the
// "telemetry" schema declared in deviceplatform.Manifest(). message_id and
// timestamp are not included here - they're added by envelope.Builder, which
// also validates the result against that schema before publishing.
func StatusFields(status Status) (json.RawMessage, error) {
	if status.Timestamp.IsZero() {
		return nil, fmt.Errorf("status timestamp is required")
	}
	fields := struct {
		CPUPercent      float64  `json:"cpu_pct"`
		MemoryUsedMiB   uint64   `json:"memory_used_mb"`
		MemoryTotalMiB  uint64   `json:"memory_total_mb"`
		DiskUsedPercent float64  `json:"disk_used_pct"`
		Load1           float64  `json:"load_1"`
		TemperatureC    *float64 `json:"temperature_c,omitempty"`
		UptimeSeconds   uint64   `json:"uptime_s"`
	}{
		CPUPercent:      status.CPUPercent,
		MemoryUsedMiB:   status.MemoryUsedMiB,
		MemoryTotalMiB:  status.MemoryTotalMiB,
		DiskUsedPercent: status.DiskUsedPercent,
		Load1:           status.Load1,
		TemperatureC:    status.TemperatureC,
		UptimeSeconds:   status.UptimeSeconds,
	}
	return json.Marshal(fields)
}
