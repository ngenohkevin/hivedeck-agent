# Hivedeck Agent

Server monitoring and management agent for Raspberry Pi (expandable to VPS).

## Overview

Hivedeck Agent is a lightweight Go-based agent that provides:

- **System Metrics** - CPU, memory, disk, and network monitoring
- **Process Management** - List and manage running processes
- **Service Management** - Control systemd services
- **Log Streaming** - Real-time log viewing via SSE
- **Docker Support** - Container management (optional)
- **File Browser** - Read-only file system browsing
- **Task Runner** - Execute pre-defined safe commands

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/ngenohkevin/hivedeck-agent.git
cd hivedeck-agent

# Copy and configure environment
cp .env.example .env
# Edit .env with your settings (especially API_KEY)

# Build and install
make build-arm64  # For Raspberry Pi
sudo ./scripts/install.sh
```

### First-Time Setup (No API Key)

When the agent starts without an API key, it enters **Setup Mode**:

```
⚠️  No API key configured - starting in SETUP MODE
📋 Open http://<server>:8091/setup to configure the agent
🔒 After setup, restart the agent to enable authentication
```

1. Open `http://<server-ip>:8091/setup` in your browser
2. Click **Generate API Key** to create a secure key
3. Copy the key (you'll need it for the dashboard)
4. Click **Save** to write it to `.env`
5. Restart the agent: `sudo systemctl restart hivedeck-agent`

### Configuration

Edit `.env` to configure the agent:

```env
# Required (auto-generated via setup page or manually set)
API_KEY=your-secure-api-key

# Optional
PORT=8091
HOST=0.0.0.0
LOG_LEVEL=info
DOCKER_ENABLED=true
ALLOWED_SERVICES=routerctl-agent,hivedeck-agent,docker,nginx,ssh,tailscaled
ALLOWED_PATHS=/var/log,/etc,/home,/opt,/tmp
WRITE_TIMEOUT_SECONDS=86400  # 24h for SSE connections
```

### Running

```bash
# Development
make dev

# Production (via systemd)
sudo systemctl start hivedeck-agent
sudo systemctl status hivedeck-agent
```

## API Reference

All API endpoints (except `/health`) require authentication via:
- `Authorization: Bearer <API_KEY>` header
- `?token=<API_KEY>` query parameter

### Health & Info

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check (no auth) |
| `/api/info` | GET | Server identity and version |

### System Metrics

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/metrics` | GET | All system metrics |
| `/api/metrics/cpu` | GET | CPU usage and load |
| `/api/metrics/memory` | GET | RAM and swap usage |
| `/api/metrics/disk` | GET | Disk partitions |
| `/api/metrics/network` | GET | Network interfaces |

### Process Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/processes` | GET | List processes (top N by CPU) |
| `/api/processes/:pid/kill` | POST | Kill process (allowlist only) |

Query parameters:
- `limit` - Number of processes to return (default: 50)

### Service Management

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/services` | GET | List allowed services |
| `/api/services/:name` | GET | Service status |
| `/api/services/:name/start` | POST | Start service |
| `/api/services/:name/stop` | POST | Stop service |
| `/api/services/:name/restart` | POST | Restart service |

### Logs

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/logs` | GET | SSE log stream |
| `/api/logs/query` | GET | Query logs |
| `/api/logs/:unit` | GET | Unit-specific logs |

Query parameters:
- `unit` - Filter by systemd unit
- `priority` - Log priority (0-7)
- `lines` - Number of lines (default: 100)
- `since` - Start time
- `until` - End time

### Docker (if enabled)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/docker/containers` | GET | List containers |
| `/api/docker/containers/:id` | GET | Container details |
| `/api/docker/containers/:id/start` | POST | Start container |
| `/api/docker/containers/:id/stop` | POST | Stop container |
| `/api/docker/containers/:id/restart` | POST | Restart container |
| `/api/docker/containers/:id/logs` | GET | Container logs |

### Files (Read-Only)

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/files` | GET | Directory listing |
| `/api/files/content` | GET | File content |
| `/api/files/diskusage` | GET | Disk usage info |

Query parameters:
- `path` - File or directory path

### Tasks

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/tasks` | GET | List available tasks |
| `/api/tasks/:name/run` | POST | Execute task |

Dangerous tasks require `?confirm=true`.

### Real-time Events

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/events` | GET | SSE metrics stream (24h timeout) |

### Setup & Settings

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/setup` | GET | Setup page (only in setup mode) |
| `/setup/generate` | POST | Generate new API key |
| `/setup/save` | POST | Save API key to .env |
| `/settings` | GET | Settings page (requires `?key=`) |
| `/api/settings` | GET | Get current settings |
| `/api/settings` | PUT | Update settings |
| `/api/settings/generate-key` | POST | Generate new API key |
| `/api/settings/api-key` | POST | Save new API key |

## Example Usage

```bash
# Health check
curl http://localhost:8091/health

# Get all metrics
curl -H "Authorization: Bearer $API_KEY" http://localhost:8091/api/metrics

# List processes
curl -H "Authorization: Bearer $API_KEY" http://localhost:8091/api/processes?limit=10

# Get service status
curl -H "Authorization: Bearer $API_KEY" http://localhost:8091/api/services/nginx

# Restart a service
curl -X POST -H "Authorization: Bearer $API_KEY" http://localhost:8091/api/services/nginx/restart

# Stream metrics (SSE)
curl -H "Authorization: Bearer $API_KEY" http://localhost:8091/api/events

# Run a task
curl -X POST -H "Authorization: Bearer $API_KEY" http://localhost:8091/api/tasks/df/run

# Run dangerous task
curl -X POST -H "Authorization: Bearer $API_KEY" "http://localhost:8091/api/tasks/reboot/run?confirm=true"
```

## Development

```bash
# Install dependencies
make deps

# Run tests
make test

# Run with hot reload
make dev

# Build for current platform
make build

# Build for Raspberry Pi
make build-arm64

# Deploy to Pi
make deploy-pi

# Lint code
make lint
```

## Project Structure

```
hivedeck-agent/
├── main.go                 # Entry point
├── config/
│   └── config.go           # Configuration management
├── internal/
│   ├── cache/              # In-memory caching
│   ├── docker/             # Docker management
│   ├── files/              # File browser
│   ├── process/            # Process management
│   ├── server/             # HTTP server, handlers, middleware
│   ├── system/             # System metrics
│   ├── systemd/            # Service and log management
│   └── tasks/              # Task runner
├── scripts/
│   ├── install.sh          # Installation script
│   └── uninstall.sh        # Uninstallation script
└── .github/workflows/
    └── deploy.yml          # CI/CD pipeline
```

## Pre-defined Tasks

| Name | Command | Description | Dangerous |
|------|---------|-------------|-----------|
| `apt-update` | `apt update` | Update package lists | No |
| `apt-upgrade` | `apt upgrade -y` | Upgrade packages | No |
| `df` | `df -h` | Check disk space | No |
| `free` | `free -m` | Check memory | No |
| `uptime` | `uptime` | System uptime | No |
| `who` | `who` | Logged-in users | No |
| `pi-temp` | `vcgencmd measure_temp` | Pi temperature | No |
| `reboot` | `reboot` | Reboot system | Yes |

## Tailscale Serve (HTTPS Access)

You can expose the agent with automatic SSL via Tailscale Serve:

```bash
# Enable Tailscale Serve (run on the server)
sudo tailscale serve --bg http://127.0.0.1:8091
```

This creates an HTTPS endpoint accessible within your tailnet:
```
https://<hostname>.tail<xxxxx>.ts.net/
```

Example URLs:
```bash
# Health check (no auth)
https://pi.taila26a58.ts.net/health

# Metrics (with auth)
https://pi.taila26a58.ts.net/api/metrics?token=YOUR_API_KEY

# Settings page
https://pi.taila26a58.ts.net/settings?key=YOUR_API_KEY
```

To disable:
```bash
sudo tailscale serve --https=443 off
```

## Security

- API key authentication required for all endpoints
- JWT token support for session-based auth
- Rate limiting (configurable RPS)
- Service allowlist restricts which services can be managed
- File browser restricted to allowed paths
- Task runner only executes pre-defined commands
- CORS configuration for frontend access

## CI/CD

The agent deploys automatically via GitHub Actions when pushed to `main`:

1. Runs tests
2. Builds for ARM64
3. Connects to Tailscale
4. Deploys to Raspberry Pi
5. Verifies health endpoint

Required secrets:
- `TS_OAUTH_CLIENT_ID` - Tailscale OAuth client ID
- `TS_OAUTH_SECRET` - Tailscale OAuth secret

## License

MIT
