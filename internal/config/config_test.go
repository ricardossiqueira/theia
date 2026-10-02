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
	if cfg.Device.StorageDir != "/var/lib/orangepi-monitor" || cfg.Device.HTTPPort != 8090 || cfg.Monitor.Interval.TimeDuration().Seconds() != 2 {
		t.Fatalf("config = %#v", cfg)
	}
}

func TestParseAllowsOmittedPairingWindow(t *testing.T) {
	data := strings.Replace(validConfig, "  pairing_window: 10m\n", "", 1)
	cfg, err := Parse([]byte(data))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Device.PairingWindow != 0 {
		t.Fatalf("PairingWindow = %v, want 0 (defers to the library default)", cfg.Device.PairingWindow)
	}
}

func TestParseRejectsInvalidConfig(t *testing.T) {
	tests := []struct{ name, replace, want string }{
		{"missing storage_dir", "  storage_dir: /var/lib/orangepi-monitor\n", ""},
		{"missing http_port", "  http_port: 8090\n", ""},
		{"negative pairing_window", "pairing_window: 10m", "pairing_window: -1s"},
		{"invalid interval", "interval: 2s", "interval: 0s"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := strings.Replace(validConfig, test.replace, test.want, 1)
			if _, err := Parse([]byte(data)); err == nil {
				t.Fatal("Parse() error = nil")
			}
		})
	}
}

const validConfig = `device:
  storage_dir: /var/lib/orangepi-monitor
  http_port: 8090
  pairing_window: 10m
monitor:
  interval: 2s
`
