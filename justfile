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
    go build -ldflags "-X main.version=$(git rev-parse --short HEAD)" -o bin/orangepi-monitor ./cmd/orangepi-monitor

# Validate only the declarative monitor configuration; this does not need a
# broker connection or Linux metric interfaces.
validate config=config:
    go run ./cmd/orangepi-monitor validate --config {{config}}

# Run the monitor in the foreground. On first run it waits, unregistered, for
# the gateway to discover, pair and provision it - see deploy/README.md.
run config=config:
    go run ./cmd/orangepi-monitor run --config {{config}}

# Deployment is now a Podman container (Quadlet) built on hosted CI and
# pulled by the device - see orangepi-deploy/README.md for the bootstrap
# and deploy/README.md for the device-specific commands (enable, forget,
# logs). Nothing is built or installed from this justfile on the Pi
# anymore.
