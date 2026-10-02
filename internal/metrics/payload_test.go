package metrics

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStatusFields(t *testing.T) {
	temp := 45.5
	payload, err := StatusFields(Status{
		Timestamp:       time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC),
		CPUPercent:      12.5,
		MemoryUsedMiB:   200,
		MemoryTotalMiB:  900,
		DiskUsedPercent: 30,
		Load1:           0.5,
		TemperatureC:    &temp,
		UptimeSeconds:   3600,
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields struct {
		CPUPercent     float64 `json:"cpu_pct"`
		MemoryUsedMiB  uint64  `json:"memory_used_mb"`
		MemoryTotalMiB uint64  `json:"memory_total_mb"`
		TemperatureC   float64 `json:"temperature_c"`
		UptimeSeconds  uint64  `json:"uptime_s"`
	}
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	if fields.CPUPercent != 12.5 || fields.MemoryUsedMiB != 200 || fields.TemperatureC != 45.5 || fields.UptimeSeconds != 3600 {
		t.Fatalf("fields = %#v", fields)
	}
	// message_id/timestamp are not this function's job anymore - envelope.Builder adds them.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["message_id"]; ok {
		t.Fatal("StatusFields must not include message_id")
	}
	if _, ok := raw["timestamp"]; ok {
		t.Fatal("StatusFields must not include timestamp")
	}
}

func TestStatusFieldsOmitsNilTemperature(t *testing.T) {
	payload, err := StatusFields(Status{Timestamp: time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["temperature_c"]; ok {
		t.Fatal("temperature_c must be omitted when nil")
	}
}

func TestStatusFieldsRequiresTimestamp(t *testing.T) {
	if _, err := StatusFields(Status{}); err == nil {
		t.Fatal("StatusFields() with zero timestamp error = nil")
	}
}
