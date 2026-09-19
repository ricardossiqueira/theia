package metrics

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"
)

func TestTelemetryPayload(t *testing.T) {
	payload, err := TelemetryPayload(Status{
		Timestamp: time.Date(2026, 9, 19, 3, 0, 0, 0, time.UTC), CPUPercent: 12.5, MemoryUsedMiB: 200, MemoryTotalMiB: 900,
	})
	if err != nil {
		t.Fatal(err)
	}
	var message struct {
		MessageID string  `json:"message_id"`
		Timestamp string  `json:"timestamp"`
		CPU       float64 `json:"cpu_pct"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(message.MessageID) || message.Timestamp != "2026-09-19T03:00:00Z" || message.CPU != 12.5 {
		t.Fatalf("message = %#v", message)
	}
}
