package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitor.yaml")
	if err := os.WriteFile(path, []byte(validConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"validate", "--config", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() = %d, stderr = %s", code, stderr.String())
	}
	if got := stdout.String(); got != "configuration is valid\n" {
		t.Errorf("stdout = %q", got)
	}
}

func TestRunValidateDoesNotRequireCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "monitor.yaml")
	if err := os.WriteFile(path, []byte(validConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ORANGEPI_MONITOR_MQTT_USERNAME", "")
	t.Setenv("ORANGEPI_MONITOR_MQTT_PASSWORD", "")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"validate", "--config", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() = %d, stderr = %s", code, stderr.String())
	}
}

func TestRunValidateRejectsInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(path, []byte("mqtt: ["), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"validate", "--config", path}, &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "configuration is invalid") {
		t.Fatalf("run() = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"unknown"}, &stdout, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("run() = %d, stderr = %q", code, stderr.String())
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
