#!/bin/bash
set -e

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/instvisor"
DATA_DIR="/var/lib/instvisor"
SYSTEMD_DIR="/etc/systemd/system"

echo "Uninstalling Instvisor..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (sudo)"
    exit 1
fi

# Stop and disable service
if command -v systemctl &> /dev/null; then
    if systemctl is-active --quiet instvisor; then
        echo "Stopping instvisor service..."
        systemctl stop instvisor
    fi
    
    if systemctl is-enabled --quiet instvisor 2>/dev/null; then
        echo "Disabling instvisor service..."
        systemctl disable instvisor
    fi
    
    if [ -f "$SYSTEMD_DIR/instvisor.service" ]; then
        echo "Removing systemd service..."
        rm "$SYSTEMD_DIR/instvisor.service"
        systemctl daemon-reload
    fi
fi

# Remove binaries
echo "Removing binaries..."
rm -f "$INSTALL_DIR/instvisor-agent"
rm -f "$INSTALL_DIR/instvisor-analyze"

# Ask about data and config
read -p "Remove configuration files? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "$CONFIG_DIR"
fi

read -p "Remove data directory (all metrics will be lost)? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf "$DATA_DIR"
fi

echo "Uninstallation complete!"