package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/ricardossiqueira/athena-go/runtime"
	"github.com/ricardossiqueira/athena-go/session"

	"github.com/ricardossiqueira/orangepi-monitor/internal/config"
	"github.com/ricardossiqueira/orangepi-monitor/internal/deviceplatform"
	"github.com/ricardossiqueira/orangepi-monitor/internal/metrics"
	"github.com/ricardossiqueira/orangepi-monitor/internal/sdnotify"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "validate" {
		return runValidate(args[1:], stdout, stderr)
	}
	if len(args) > 0 && args[0] == "forget" {
		return runForget(args[1:], stdout, stderr)
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

// runForget clears any persisted provisioning, forcing the device back into
// its pairing window on the next run. It never touches the identity file:
// device_uid and the identity key stay stable across re-registration.
func runForget(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("forget", flag.ContinueOnError)
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
	storage := deviceplatform.FileStorage{Dir: cfg.Device.StorageDir}
	if err := storage.ClearProvisioning(); err != nil {
		fmt.Fprintf(stderr, "forget provisioning failed: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "provisioning cleared; device will re-enter its pairing window on next start")
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
	logger := slog.New(slog.NewTextHandler(stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := os.MkdirAll(cfg.Device.StorageDir, 0o700); err != nil {
		logger.Error("storage directory setup failed", "error", err)
		return 1
	}
	storage := deviceplatform.FileStorage{Dir: cfg.Device.StorageDir}
	rt, err := runtime.New(runtime.Config{
		Manifest:        deviceplatform.Manifest(),
		FirmwareVersion: version,
		Storage:         storage,
		PairingWindow:   cfg.Device.PairingWindow.TimeDuration(),
	})
	if err != nil {
		logger.Error("device platform setup failed", "error", err)
		return 1
	}

	httpServer := &http.Server{Addr: fmt.Sprintf(":%d", cfg.Device.HTTPPort), Handler: deviceplatform.NewHandler(rt)}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("device-info server failed", "error", err)
		}
	}()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	announcement, err := rt.NetworkReady(cfg.Device.HTTPPort)
	if err != nil {
		logger.Error("mDNS announcement setup failed", "error", err)
		return 1
	}
	mdnsServer, err := deviceplatform.Announce(deviceplatform.Announcement{Service: announcement.Service, Port: announcement.Port, TXT: announcement.TXT})
	if err != nil {
		logger.Error("mDNS announcement failed", "error", err)
		return 1
	}
	defer mdnsServer.Shutdown()

	logger.Info("waiting for provisioning", "device_uid", rt.Identity().DeviceUID())
	provisioning, ok := waitForProvisioning(ctx, rt)
	if !ok {
		return 0
	}
	logger.Info("provisioned", "device_id", provisioning.DeviceID)

	username, password := provisioning.MQTT.Username, provisioning.MQTT.Password
	options := paho.NewClientOptions().
		AddBroker(fmt.Sprintf("mqtt://%s:%d", provisioning.MQTT.Host, provisioning.MQTT.Port)).
		SetClientID(provisioning.DeviceID).
		SetUsername(username).
		SetPassword(password).
		SetCleanSession(false).
		SetAutoReconnect(true).
		SetConnectRetry(true)
	client := paho.NewClient(options)
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
	// publish returns its error (rather than only logging it) so the first
	// call below can gate the systemd readiness notification: subsequent
	// ticker-driven calls still just log, matching the previous behavior.
	publish := func() error {
		status, err := collector.Collect()
		if err != nil {
			logger.Error("metrics collection failed", "error", err)
			return err
		}
		fields, err := metrics.StatusFields(status)
		if err != nil {
			logger.Error("telemetry encoding failed", "error", err)
			return err
		}
		message, err := rt.Telemetry(fields)
		if err != nil {
			logger.Error("telemetry envelope failed", "error", err)
			return err
		}
		if err := wait(ctx, client.Publish(message.Topic, message.QoS, message.Retained, message.Payload)); err != nil {
			if ctx.Err() == nil {
				logger.Error("telemetry publish failed", "error", err)
			}
			return err
		}
		return nil
	}
	// Only the first successful publish - MQTT connected and real telemetry
	// delivered - signals readiness. A Type=notify unit then blocks
	// `systemctl start`/`restart` on this, instead of declaring success the
	// moment the process merely starts (see deploy/orangepi-monitor.service).
	if err := publish(); err == nil {
		if err := sdnotify.Ready(); err != nil {
			logger.Error("systemd readiness notification failed", "error", err)
		}
	}
	ticker := time.NewTicker(cfg.Monitor.Interval.TimeDuration())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return 0
		case <-ticker.C:
			_ = publish()
		}
	}
}

// waitForProvisioning polls Runtime.Provisioning() until the gateway
// completes pairing/provisioning or ctx is cancelled. Polling (rather than a
// callback threaded through the HTTP handler) is adequate here: pairing
// happens at human/operator pace, not in a hot path.
func waitForProvisioning(ctx context.Context, rt *runtime.Runtime) (session.Provisioning, bool) {
	if p, ok := rt.Provisioning(); ok {
		return p, true
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return session.Provisioning{}, false
		case <-ticker.C:
			if p, ok := rt.Provisioning(); ok {
				return p, true
			}
		}
	}
}

func printUsage(stderr io.Writer) {
	fmt.Fprintln(stderr, "usage: orangepi-monitor [validate|run|forget] --config <path>")
}

func wait(ctx context.Context, token paho.Token) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-token.Done():
		return token.Error()
	}
}
