// Package deviceplatform adapts orangepi-monitor to iot-device-core-go: the
// manifest declaration, file-based storage, HTTP inspection/pairing server
// and mDNS announcement that make it a real device platform v2 device.
package deviceplatform

import (
	"encoding/json"

	"github.com/ricardossiqueira/iot-device-core-go/manifest"
)

// Manifest matches contracts/device-v2/fixtures/orangepi-manifest.json and
// the fields internal/metrics/payload.go already publishes.
func Manifest() manifest.Manifest {
	field := func(t string, required bool) manifest.Field { return manifest.Field{Type: t, Required: required} }
	return manifest.Manifest{
		SchemaVersion:   manifest.SchemaVersion,
		ManifestID:      "orangepi-monitor",
		DisplayName:     "Orange Pi Monitor",
		Model:           "orangepi-monitor",
		ProtocolVersion: 1,
		MQTT: manifest.MQTT{
			Publish: []manifest.Output{{
				Channel: "telemetry",
				Schema: mustMarshalFields(map[string]manifest.Field{
					"cpu_pct":         field("number", true),
					"memory_used_mb":  field("integer", true),
					"memory_total_mb": field("integer", true),
					"disk_used_pct":   field("number", true),
					"load_1":          field("number", true),
					"temperature_c":   field("number", false),
					"uptime_s":        field("integer", true),
				}),
			}},
		},
	}
}

func mustMarshalFields(fields map[string]manifest.Field) json.RawMessage {
	b, err := json.Marshal(fields)
	if err != nil {
		panic(err)
	}
	return b
}
