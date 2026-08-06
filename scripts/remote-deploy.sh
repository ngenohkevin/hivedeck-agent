#!/usr/bin/env bash
#
# Installs or updates hivedeck-agent on a remote host.
#
# CI pipes this over ssh, with the new binary already staged as
# "$DIR/hivedeck-agent.new":
#
#   ssh root@HOST "DIR=/opt/hivedeck-agent ... bash -s" < scripts/remote-deploy.sh
#
# It is idempotent: .env and the unit file are only created when missing, so
# hand-tuned settings on a host are never overwritten by a deploy.

set -euo pipefail

DIR="${DIR:-/opt/hivedeck-agent}"
PORT="${PORT:-8091}"
DOCKER_ENABLED="${DOCKER_ENABLED:-true}"
ALLOWED_SERVICES="${ALLOWED_SERVICES:-docker,ssh,tailscaled,hivedeck-agent}"
ALLOWED_PATHS="${ALLOWED_PATHS:-/var/log,/etc,/opt}"
# Bind the API to the host's Tailscale address rather than 0.0.0.0. Set to
# "false" only for a host that must serve on another interface.
BIND_TAILSCALE="${BIND_TAILSCALE:-true}"

UNIT=/etc/systemd/system/hivedeck-agent.service

cd "$DIR"

if [ ! -f hivedeck-agent.new ]; then
  echo "error: no staged binary at $DIR/hivedeck-agent.new" >&2
  exit 1
fi

# --- bind address ------------------------------------------------------------
# SECURITY: this agent exposes metrics, journald, the file browser and
# service/container control. It has no business on a public interface. Prefer
# the Tailscale address; fall back to loopback rather than 0.0.0.0 so that a
# missing or late tailscaled can never silently publish it to the internet.
# (See SECURITY.md 2026-06-08 — this agent was reachable on public IPv6.)
pick_bind_host() {
  local ip=""
  if [ "$BIND_TAILSCALE" = "true" ]; then
    ip="$(tailscale ip -4 2>/dev/null | head -n1 || true)"
  fi
  if [ -n "$ip" ]; then
    echo "$ip"
  else
    echo "127.0.0.1"
  fi
}

# --- .env --------------------------------------------------------------------
if [ ! -f .env ]; then
  if ! command -v openssl >/dev/null 2>&1; then
    echo "error: openssl is required to generate the initial API key" >&2
    exit 1
  fi

  api_key="$(openssl rand -hex 32)"
  jwt_secret="$(openssl rand -hex 32)"
  bind_host="$(pick_bind_host)"

  ( umask 077; cat > .env <<ENVEOF
PORT=${PORT}
HOST=${bind_host}
API_KEY=${api_key}
JWT_SECRET=${jwt_secret}
ALLOWED_ORIGINS=*
RATE_LIMIT_RPS=100
DOCKER_ENABLED=${DOCKER_ENABLED}
LOG_LEVEL=info
ALLOWED_SERVICES=${ALLOWED_SERVICES}
ALLOWED_PATHS=${ALLOWED_PATHS}
ENVEOF
  )

  echo "Created .env with a freshly generated API key, bound to ${bind_host}:${PORT}."
  echo "Register this host in the Hivedeck dashboard with:"
  echo "  ssh root@$(hostname -s) \"grep '^API_KEY=' ${DIR}/.env\""
fi

# Always enforce; the file holds an API key and a JWT secret.
chmod 600 .env

# --- systemd unit ------------------------------------------------------------
if [ ! -f "$UNIT" ]; then
  after="network-online.target tailscaled.service"
  wants="network-online.target tailscaled.service"
  if [ "$DOCKER_ENABLED" = "true" ]; then
    after="$after docker.service"
    wants="$wants docker.service"
  fi

  cat > "$UNIT" <<SVCEOF
[Unit]
Description=Hivedeck Agent - Server Monitoring Agent
Documentation=https://github.com/ngenohkevin/hivedeck-agent
After=${after}
Wants=${wants}

[Service]
Type=simple
User=root
WorkingDirectory=${DIR}
ExecStart=${DIR}/hivedeck-agent
Restart=always
RestartSec=5
EnvironmentFile=${DIR}/.env

[Install]
WantedBy=multi-user.target
SVCEOF

  systemctl daemon-reload
  systemctl enable hivedeck-agent
  echo "Created and enabled systemd service"
fi

# --- swap the binary ---------------------------------------------------------
if [ -f hivedeck-agent ]; then
  cp -f hivedeck-agent hivedeck-agent.prev
fi

systemctl stop hivedeck-agent || true
mv hivedeck-agent.new hivedeck-agent
chmod 755 hivedeck-agent
systemctl start hivedeck-agent

# --- verify, roll back on failure --------------------------------------------
health_host="$(grep -m1 '^HOST=' .env | cut -d= -f2-)"
health_port="$(grep -m1 '^PORT=' .env | cut -d= -f2-)"
[ -n "$health_host" ] && [ "$health_host" != "0.0.0.0" ] || health_host=127.0.0.1
[ -n "$health_port" ] || health_port="$PORT"

healthy=0
for _ in $(seq 1 15); do
  if curl -fsS --max-time 3 "http://${health_host}:${health_port}/health" >/dev/null 2>&1; then
    healthy=1
    break
  fi
  sleep 1
done

if [ "$healthy" != "1" ]; then
  echo "error: agent failed its health check after the update" >&2
  journalctl -u hivedeck-agent -n 40 --no-pager || true

  if [ -f hivedeck-agent.prev ]; then
    echo "rolling back to the previous binary" >&2
    systemctl stop hivedeck-agent || true
    mv -f hivedeck-agent.prev hivedeck-agent
    systemctl start hivedeck-agent
  fi
  exit 1
fi

rm -f hivedeck-agent.prev
echo "hivedeck-agent healthy on ${health_host}:${health_port} — $(./hivedeck-agent -version 2>/dev/null || echo 'version unknown')"
