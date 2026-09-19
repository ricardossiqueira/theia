// Package config loads the independent Orange Pi monitor configuration.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MQTT    MQTT    `yaml:"mqtt"`
	Monitor Monitor `yaml:"monitor"`
}

type MQTT struct {
	URL         string `yaml:"url"`
	ClientID    string `yaml:"client_id"`
	UsernameEnv string `yaml:"username_env"`
	PasswordEnv string `yaml:"password_env"`
}

type Monitor struct {
	DeviceID string   `yaml:"device_id"`
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
	parsed, err := url.Parse(c.MQTT.URL)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "mqtt" && parsed.Scheme != "mqtts") || parsed.Port() == "" {
		return errors.New("mqtt.url must be a mqtt or mqtts URL with host and port")
	}
	if strings.TrimSpace(c.MQTT.ClientID) == "" {
		return errors.New("mqtt.client_id is required")
	}
	if strings.TrimSpace(c.MQTT.UsernameEnv) == "" || strings.TrimSpace(c.MQTT.PasswordEnv) == "" {
		return errors.New("mqtt.username_env and mqtt.password_env are required")
	}
	if strings.TrimSpace(c.Monitor.DeviceID) == "" {
		return errors.New("monitor.device_id is required")
	}
	if c.Monitor.Interval.TimeDuration() <= 0 {
		return errors.New("monitor.interval must be greater than zero")
	}
	return nil
}
