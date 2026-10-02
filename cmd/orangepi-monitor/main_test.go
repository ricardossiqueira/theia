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

func TestRunValidateRejectsInvalidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(path, []byte("device: ["), 0o600); err != nil {
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

func TestRunForgetClearsProvisioning(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "monitor.yaml")
	storageDir := filepath.Join(dir, "storage")
	if err := os.MkdirAll(storageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	config := strings.Replace(validConfig, "storage_dir: /var/lib/orangepi-monitor", "storage_dir: "+filepath.ToSlash(storageDir), 1)
	if err := os.WriteFile(path, []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storageDir, "provisioning.json"), []byte(`{"device_id":"d"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if code := run([]string{"forget", "--config", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(storageDir, "provisioning.json")); !os.IsNotExist(err) {
		t.Fatalf("provisioning.json still exists: %v", err)
	}
}

func TestRunForgetRequiresValidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(path, []byte("device: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{"forget", "--config", path}, &stdout, &stderr); code != 1 {
		t.Fatalf("run() = %d, stderr = %q", code, stderr.String())
	}
}

const validConfig = `device:
  storage_dir: /var/lib/orangepi-monitor
  http_port: 8090
  pairing_window: 10m
monitor:
  interval: 2s
`
