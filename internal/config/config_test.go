package config

import (
	"strings"
	"testing"
)

func TestParseValidConfig(t *testing.T) {
	cfg, err := Parse([]byte(validConfig))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Monitor.DeviceID != "orangepi-monitor" || cfg.Monitor.Interval.TimeDuration().Seconds() != 2 {
		t.Fatalf("config = %#v", cfg)
	}
}

func TestParseRejectsInvalidConfig(t *testing.T) {
	tests := []struct{ name, replace, want string }{
		{"missing credentials", "  password_env: ORANGEPI_MONITOR_MQTT_PASSWORD\n", "password_env"},
		{"invalid interval", "interval: 2s", "interval: 0s"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := validConfig
			if test.name == "invalid interval" {
				data = strings.Replace(data, test.replace, test.want, 1)
			} else {
				data = strings.Replace(data, test.replace, "", 1)
			}
			if _, err := Parse([]byte(data)); err == nil {
				t.Fatal("Parse() error = nil")
			}
		})
	}
}

const validConfig = `mqtt:
  url: mqtt://127.0.0.1:1883
  client_id: orangepi-monitor
  username_env: ORANGEPI_MONITOR_MQTT_USERNAME
  password_env: ORANGEPI_MONITOR_MQTT_PASSWORD
monitor:
  device_id: orangepi-monitor
  interval: 2s
`
