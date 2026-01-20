#!/bin/bash
set -e

# Hivedeck Agent Uninstallation Script

SERVICE_NAME="hivedeck-agent"
INSTALL_DIR="/home/ngenoh/dev/backend/hivedeck-agent"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Uninstalling Hivedeck Agent...${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}Please run as root${NC}"
    exit 1
fi

# Stop the service if running
if systemctl is-active --quiet ${SERVICE_NAME}; then
    echo "Stopping ${SERVICE_NAME}..."
    systemctl stop ${SERVICE_NAME}
fi

# Disable the service
if systemctl is-enabled --quiet ${SERVICE_NAME} 2>/dev/null; then
    echo "Disabling ${SERVICE_NAME}..."
    systemctl disable ${SERVICE_NAME}
fi

# Remove systemd service file
if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
    rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
    echo "Removed systemd service file"
fi

# Reload systemd
systemctl daemon-reload

echo -e "${GREEN}Hivedeck Agent uninstalled successfully${NC}"
echo -e "${YELLOW}Note: The installation directory ($INSTALL_DIR) was not removed.${NC}"
echo -e "To fully remove, run: ${YELLOW}rm -rf $INSTALL_DIR${NC}"
