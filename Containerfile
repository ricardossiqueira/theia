# Built on GitHub-hosted CI (linux/arm64 via QEMU), never on the Orange Pi
# itself - see .github/workflows/ci.yml's publish-image job and
# orangepi-deploy/README.md for why native builds don't happen on the
# device anymore.
FROM golang:1.25-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG REVISION=dev
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-X main.version=${REVISION}" -o /out/orangepi-monitor ./cmd/orangepi-monitor

# :nonroot runs as UID/GID 65532 - the host path bind-mounted at
# /var/lib/orangepi-monitor must be chown'd to 65532:65532 (see
# orangepi-deploy/README.md's bootstrap).
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/orangepi-monitor /usr/local/bin/orangepi-monitor
ENTRYPOINT ["/usr/local/bin/orangepi-monitor"]
