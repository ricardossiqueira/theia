package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"

	"github.com/ricardossiqueira/orangepi-monitor/internal/config"
	"github.com/ricardossiqueira/orangepi-monitor/internal/metrics"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "path to the YAML configuration")
	flag.Parse()
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	username, password := os.Getenv(cfg.MQTT.UsernameEnv), os.Getenv(cfg.MQTT.PasswordEnv)
	if username == "" || password == "" {
		fmt.Fprintln(os.Stderr, "configured MQTT credentials are not set")
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	options := paho.NewClientOptions().AddBroker(cfg.MQTT.URL).SetClientID(cfg.MQTT.ClientID).SetUsername(username).SetPassword(password).SetCleanSession(false).SetAutoReconnect(true).SetConnectRetry(true)
	client := paho.NewClient(options)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := wait(ctx, client.Connect()); err != nil {
		logger.Error("MQTT connection failed", "error", err)
		os.Exit(1)
	}
	defer client.Disconnect(250)
	collector, err := metrics.NewLinuxCollector()
	if err != nil {
		logger.Error("collector setup failed", "error", err)
		os.Exit(1)
	}
	topic := "devices/" + cfg.Monitor.DeviceID + "/telemetry"
	publish := func() {
		status, err := collector.Collect()
		if err != nil {
			logger.Error("metrics collection failed", "error", err)
			return
		}
		payload, err := metrics.TelemetryPayload(status)
		if err != nil {
			logger.Error("telemetry encoding failed", "error", err)
			return
		}
		if err := wait(ctx, client.Publish(topic, 1, false, payload)); err != nil && ctx.Err() == nil {
			logger.Error("telemetry publish failed", "error", err)
		}
	}
	publish()
	ticker := time.NewTicker(cfg.Monitor.Interval.TimeDuration())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			publish()
		}
	}
}

func wait(ctx context.Context, token paho.Token) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		return token.Error()
	}
}
