// Package metrics collects host metrics and builds canonical telemetry payloads.
package metrics

import (
	"crypto/rand"
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

func TelemetryPayload(status Status) ([]byte, error) {
	if status.Timestamp.IsZero() {
		return nil, fmt.Errorf("status timestamp is required")
	}
	id, err := uuidV4()
	if err != nil {
		return nil, err
	}
	payload := struct {
		MessageID       string   `json:"message_id"`
		Timestamp       string   `json:"timestamp"`
		CPUPercent      float64  `json:"cpu_pct"`
		MemoryUsedMiB   uint64   `json:"memory_used_mb"`
		MemoryTotalMiB  uint64   `json:"memory_total_mb"`
		DiskUsedPercent float64  `json:"disk_used_pct"`
		Load1           float64  `json:"load_1"`
		TemperatureC    *float64 `json:"temperature_c,omitempty"`
		UptimeSeconds   uint64   `json:"uptime_s"`
	}{
		MessageID:       id,
		Timestamp:       status.Timestamp.UTC().Format(time.RFC3339),
		CPUPercent:      status.CPUPercent,
		MemoryUsedMiB:   status.MemoryUsedMiB,
		MemoryTotalMiB:  status.MemoryTotalMiB,
		DiskUsedPercent: status.DiskUsedPercent,
		Load1:           status.Load1,
		TemperatureC:    status.TemperatureC,
		UptimeSeconds:   status.UptimeSeconds,
	}
	return json.Marshal(payload)
}

func uuidV4() (string, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return "", err
	}
	id[6] = (id[6] & 0x0f) | 0x40
	id[8] = (id[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", id[0:4], id[4:6], id[6:8], id[8:10], id[10:16]), nil
}
