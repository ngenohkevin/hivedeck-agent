#!/bin/bash
set -e

# Hivedeck Agent Installation Script

INSTALL_DIR="/home/ngenoh/dev/backend/hivedeck-agent"
SERVICE_NAME="hivedeck-agent"
BINARY_NAME="hivedeck-agent"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Installing Hivedeck Agent...${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Please run as root${NC}"
    exit 1
fi

# Create install directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    mkdir -p "$INSTALL_DIR"
    echo -e "${GREEN}Created directory: $INSTALL_DIR${NC}"
fi

# Copy binary if it exists in current directory
if [ -f "./$BINARY_NAME" ]; then
    cp "./$BINARY_NAME" "$INSTALL_DIR/"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    echo -e "${GREEN}Copied binary to $INSTALL_DIR${NC}"
elif [ -f "./${BINARY_NAME}-linux-arm64" ]; then
    cp "./${BINARY_NAME}-linux-arm64" "$INSTALL_DIR/$BINARY_NAME"
    chmod +x "$INSTALL_DIR/$BINARY_NAME"
    echo -e "${GREEN}Copied ARM64 binary to $INSTALL_DIR${NC}"
fi

# Create .env file if it doesn't exist
if [ ! -f "$INSTALL_DIR/.env" ]; then
    if [ -f "./.env.example" ]; then
        cp "./.env.example" "$INSTALL_DIR/.env"
        echo -e "${YELLOW}Created .env file from example. Please edit $INSTALL_DIR/.env with your settings.${NC}"
    else
        cat > "$INSTALL_DIR/.env" << 'EOF'
PORT=8091
HOST=0.0.0.0
API_KEY=change-me-to-a-secure-key
JWT_SECRET=change-me-to-a-secure-secret
ALLOWED_ORIGINS=*
RATE_LIMIT_RPS=100
DOCKER_ENABLED=false
LOG_LEVEL=info
ALLOWED_SERVICES=routerctl-agent,hivedeck-agent,docker,nginx,ssh,tailscaled
EOF
        echo -e "${YELLOW}Created default .env file. Please edit $INSTALL_DIR/.env with your settings.${NC}"
    fi
fi

# Create systemd service file
cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=Hivedeck Agent - Server Monitoring Agent
Documentation=https://github.com/ngenohkevin/hivedeck-agent
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=always
RestartSec=5
EnvironmentFile=$INSTALL_DIR/.env

# Security hardening
NoNewPrivileges=false
ProtectSystem=false
ProtectHome=false
ReadOnlyPaths=/etc /usr
ReadWritePaths=/var/log /tmp

# Resource limits
LimitNOFILE=65535
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
EOF

echo -e "${GREEN}Created systemd service file${NC}"

# Reload systemd
systemctl daemon-reload

# Enable and start service
systemctl enable ${SERVICE_NAME}
systemctl start ${SERVICE_NAME}

echo -e "${GREEN}Service enabled and started${NC}"

# Check status
sleep 2
if systemctl is-active --quiet ${SERVICE_NAME}; then
    echo -e "${GREEN}Hivedeck Agent is running!${NC}"
    echo -e "Check status: ${YELLOW}systemctl status ${SERVICE_NAME}${NC}"
    echo -e "View logs: ${YELLOW}journalctl -u ${SERVICE_NAME} -f${NC}"
    echo -e "Test health: ${YELLOW}curl http://localhost:8091/health${NC}"
else
    echo -e "${RED}Service failed to start. Check logs:${NC}"
    journalctl -u ${SERVICE_NAME} -n 20 --no-pager
    exit 1
fi
