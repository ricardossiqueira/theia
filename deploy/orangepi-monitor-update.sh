#!/usr/bin/env bash
set -Eeuo pipefail

# This script is run as root by systemd. Git access and compilation run as the
# unprivileged repository owner; only installation and service control require
# root. Local configuration and credentials are deliberately never overwritten.
repository="/home/orangepi/orangepi-monitor"
deploy_user="orangepi"
deploy_home="/home/orangepi"
branch="main"
# This must not share the monitor's own configuration directory. Deployment
# state is privileged root-owned data and the monitor never needs to write it.
state_dir="/var/lib/orangepi-monitor-update"
state_file="$state_dir/deployed-revision"
binary="/usr/local/bin/orangepi-monitor"
unit="/etc/systemd/system/orangepi-monitor.service"

log() { logger -t orangepi-monitor-update -- "$*"; }
fail() { log "update skipped: $*"; exit 1; }
run_as_deploy_user() {
  # GOPRIVATE: github.com/ricardossiqueira/iot-device-core-go is a private
  # sibling module this repo depends on directly - go must fetch it straight
  # from git (via the deploy-user SSH config, see deploy/README.md) instead
  # of the public module proxy/checksum database, which can't see it anyway.
  runuser -u "$deploy_user" -- env HOME="$deploy_home" GOPRIVATE="github.com/ricardossiqueira/iot-device-core-go" "$@"
}

[[ -d "$repository/.git" ]] || fail "repository not found at $repository"
install -d -m 0700 -o root -g root "$state_dir"

exec 9>"$state_dir/update.lock"
flock -n 9 || { log "another update is already running"; exit 0; }

if [[ -n "$(run_as_deploy_user git -C "$repository" status --porcelain --untracked-files=all)" ]]; then
  fail "repository has local changes; refusing to overwrite them"
fi

run_as_deploy_user git -C "$repository" fetch --quiet origin "$branch" || fail "git fetch failed"
remote_revision="$(run_as_deploy_user git -C "$repository" rev-parse "origin/$branch")" || fail "remote branch not found"
current_revision="$(run_as_deploy_user git -C "$repository" rev-parse HEAD)" || fail "cannot read local revision"

run_as_deploy_user git -C "$repository" merge-base --is-ancestor "$current_revision" "$remote_revision" || fail "local revision is not an ancestor of origin/$branch"
if [[ "$current_revision" != "$remote_revision" ]]; then
  run_as_deploy_user git -C "$repository" merge --ff-only "origin/$branch" || fail "fast-forward merge failed"
  current_revision="$(run_as_deploy_user git -C "$repository" rev-parse HEAD)"
fi
[[ "$current_revision" == "$remote_revision" ]] || fail "local revision does not match origin/$branch"

deployed_revision="$(cat "$state_file" 2>/dev/null || true)"
if [[ "$current_revision" == "$deployed_revision" ]]; then
  log "already running revision $current_revision"
  exit 0
fi

candidate="$repository/bin/orangepi-monitor.candidate"
rm -f "$candidate"
trap 'rm -f "$candidate"' EXIT
run_as_deploy_user go -C "$repository" test ./... || fail "tests failed for $current_revision"
run_as_deploy_user go -C "$repository" vet ./... || fail "vet failed for $current_revision"
run_as_deploy_user go -C "$repository" build -o "$candidate" ./cmd/orangepi-monitor || fail "build failed for $current_revision"

candidate_binary="/usr/local/lib/orangepi-monitor/orangepi-monitor.candidate"
backup_binary="/usr/local/lib/orangepi-monitor/orangepi-monitor.previous"
backup_unit="/usr/local/lib/orangepi-monitor/orangepi-monitor.service.previous"
# The service account needs to execute the root-owned candidate for offline
# configuration validation, but cannot modify this directory.
install -d -m 0750 -o root -g orangepi-monitor /usr/local/lib/orangepi-monitor
install -m 0755 "$candidate" "$candidate_binary"
runuser -u orangepi-monitor -- "$candidate_binary" validate --config /etc/orangepi-monitor/config.yaml || fail "installed configuration is invalid"

had_binary=false
had_unit=false
if [[ -f "$binary" ]]; then
  install -m 0755 "$binary" "$backup_binary"
  had_binary=true
fi
if [[ -f "$unit" ]]; then
  install -m 0644 "$unit" "$backup_unit"
  had_unit=true
fi

install -m 0755 "$candidate_binary" "$binary"
install -m 0644 "$repository/deploy/orangepi-monitor.service" "$unit"
install -m 0755 "$repository/deploy/orangepi-monitor-update.sh" /usr/local/sbin/orangepi-monitor-update
install -m 0644 "$repository/deploy/orangepi-monitor-update.service" /etc/systemd/system/orangepi-monitor-update.service
install -m 0644 "$repository/deploy/orangepi-monitor-update.timer" /etc/systemd/system/orangepi-monitor-update.timer
systemctl daemon-reload

if ! systemctl restart orangepi-monitor.service; then
  log "new revision failed to start; restoring the previous service"
  if [[ "$had_binary" == true ]]; then
    install -m 0755 "$backup_binary" "$binary"
  fi
  if [[ "$had_unit" == true ]]; then
    install -m 0644 "$backup_unit" "$unit"
  fi
  systemctl daemon-reload
  systemctl restart orangepi-monitor.service || log "rollback could not restart the previous service"
  exit 1
fi
sleep 3
if ! systemctl is-active --quiet orangepi-monitor.service; then
  log "new revision exited after startup; restoring the previous service"
  if [[ "$had_binary" == true ]]; then
    install -m 0755 "$backup_binary" "$binary"
  fi
  if [[ "$had_unit" == true ]]; then
    install -m 0644 "$backup_unit" "$unit"
  fi
  systemctl daemon-reload
  systemctl restart orangepi-monitor.service || log "rollback could not restart the previous service"
  exit 1
fi

printf '%s\n' "$current_revision" >"$state_file"
rm -f "$backup_binary" "$backup_unit" "$candidate_binary"
log "deployed revision $current_revision"
