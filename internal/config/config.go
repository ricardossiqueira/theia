// Package config loads the independent Orange Pi monitor configuration.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Device  Device  `yaml:"device"`
	Monitor Monitor `yaml:"monitor"`
}

// Device configures the device platform v2 adapter: where the identity and
// provisioned MQTT credentials are persisted, and the HTTP port the gateway
// uses for discovery, pairing and provisioning. MQTT host/port/credentials
// are deliberately absent here - they only ever come from the gateway's
// provisioning payload, never from this file.
type Device struct {
	StorageDir    string   `yaml:"storage_dir"`
	HTTPPort      uint16   `yaml:"http_port"`
	PairingWindow Duration `yaml:"pairing_window"`
}

type Monitor struct {
	Interval Duration `yaml:"interval"`
}

type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.ScalarNode || value.Tag != "!!str" {
		return errors.New("duration must be a string")
	}
	parsed, err := time.ParseDuration(value.Value)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", value.Value, err)
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) TimeDuration() time.Duration { return time.Duration(d) }

func Load(path string) (Config, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Config{}, fmt.Errorf("read configuration %q: %w", path, err)
	}
	return Parse(data)
}

func Parse(data []byte) (Config, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return Config{}, errors.New("configuration is empty")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var cfg Config
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode YAML configuration: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Config{}, errors.New("configuration must contain one YAML document")
		}
		return Config{}, fmt.Errorf("decode YAML configuration: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.Device.StorageDir) == "" {
		return errors.New("device.storage_dir is required")
	}
	if c.Device.HTTPPort == 0 {
		return errors.New("device.http_port is required")
	}
	if c.Device.PairingWindow < 0 {
		return errors.New("device.pairing_window must not be negative")
	}
	if c.Monitor.Interval.TimeDuration() <= 0 {
		return errors.New("monitor.interval must be greater than zero")
	}
	return nil
}
