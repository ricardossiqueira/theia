package main

import (
	"context"
	"flag"
	"fmt"
	"io"
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
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "validate" {
		return runValidate(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "run" {
		return runMonitor(args[1:], stderr)
	}
	return runMonitor(args, stderr)
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("validate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "config/config.yaml", "path to the YAML configuration")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		printUsage(stderr)
		return 2
	}
	if _, err := config.Load(*configPath); err != nil {
		fmt.Fprintf(stderr, "configuration is invalid: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "configuration is valid")
	return 0
}

func runMonitor(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("orangepi-monitor", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "config/config.yaml", "path to the YAML configuration")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		printUsage(stderr)
		return 2
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "configuration is invalid: %v\n", err)
		return 1
	}
	username, password := os.Getenv(cfg.MQTT.UsernameEnv), os.Getenv(cfg.MQTT.PasswordEnv)
	if username == "" || password == "" {
		fmt.Fprintln(stderr, "configured MQTT credentials are not set")
		return 1
	}

	logger := slog.New(slog.NewTextHandler(stderr, nil))
	options := paho.NewClientOptions().AddBroker(cfg.MQTT.URL).SetClientID(cfg.MQTT.ClientID).SetUsername(username).SetPassword(password).SetCleanSession(false).SetAutoReconnect(true).SetConnectRetry(true)
	client := paho.NewClient(options)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := wait(ctx, client.Connect()); err != nil {
		logger.Error("MQTT connection failed", "error", err)
		return 1
	}
	defer client.Disconnect(250)
	collector, err := metrics.NewLinuxCollector()
	if err != nil {
		logger.Error("collector setup failed", "error", err)
		return 1
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
			return 0
		case <-ticker.C:
			publish()
		}
	}
}

func printUsage(stderr io.Writer) {
	fmt.Fprintln(stderr, "usage: orangepi-monitor [validate|run] --config <path>")
}

func wait(ctx context.Context, token paho.Token) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		return token.Error()
	}
}
