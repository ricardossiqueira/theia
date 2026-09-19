# Orange Pi Monitor - common development and Orange Pi deployment tasks.
# Install just: https://github.com/casey/just

# On Windows, use PowerShell for the development-only recipes.
set windows-shell := ["powershell.exe", "-NoLogo", "-NoProfile", "-Command"]

config := "config/config.yaml"

# List available recipes.
default:
    @just --list

# Format Go source files.
fmt:
    go fmt ./...

# Run the unit test suite.
test:
    go test ./...

# Run static analysis.
vet:
    go vet ./...

# Run all non-mutating local checks.
check: test vet

# Build the monitor binary for the current platform.
build:
    go build -o bin/orangepi-monitor ./cmd/orangepi-monitor

# Validate only the declarative monitor configuration; this does not need MQTT
# credentials, a broker connection, or Linux metric interfaces.
validate config=config:
    go run ./cmd/orangepi-monitor validate --config {{config}}

# Run the monitor in the foreground. MQTT credential variables must be set.
run config=config:
    go run ./cmd/orangepi-monitor run --config {{config}}

# Install or update the systemd unit on an Orange Pi. Create the environment
# file with ORANGEPI_MONITOR_MQTT_USERNAME and ORANGEPI_MONITOR_MQTT_PASSWORD
# before enabling it.
install-service config=config:
    go build -o bin/orangepi-monitor ./cmd/orangepi-monitor
    id orangepi-monitor >/dev/null 2>&1 || sudo useradd --system --user-group --no-create-home --shell /usr/sbin/nologin orangepi-monitor
    sudo install -d -m 0750 -o root -g orangepi-monitor /etc/orangepi-monitor
    sudo install -m 0755 bin/orangepi-monitor /usr/local/bin/orangepi-monitor
    sudo install -m 0640 -o root -g orangepi-monitor {{config}} /etc/orangepi-monitor/config.yaml
    sudo install -m 0644 deploy/orangepi-monitor.service /etc/systemd/system/orangepi-monitor.service
    sudo systemctl daemon-reload

# Enable the service now and on subsequent boots.
enable-service:
    sudo systemctl enable --now orangepi-monitor.service

# Display service status.
service-status:
    systemctl status orangepi-monitor.service

# Follow service logs.
service-logs:
    journalctl -u orangepi-monitor.service -f

# Install the root-owned pull-based deployment agent for an Orange Pi.
install-update-agent:
    sudo install -m 0755 deploy/orangepi-monitor-update.sh /usr/local/sbin/orangepi-monitor-update
    sudo install -m 0644 deploy/orangepi-monitor-update.service /etc/systemd/system/orangepi-monitor-update.service
    sudo install -m 0644 deploy/orangepi-monitor-update.timer /etc/systemd/system/orangepi-monitor-update.timer
    sudo systemctl daemon-reload

# Check for a new main revision every five minutes and apply it safely.
enable-update-agent:
    sudo systemctl enable --now orangepi-monitor-update.timer

# Display the schedule and last result of automatic deployment.
update-agent-status:
    systemctl status orangepi-monitor-update.timer
    systemctl status orangepi-monitor-update.service

# Follow automatic deployment logs.
update-agent-logs:
    journalctl -u orangepi-monitor-update.service -f
