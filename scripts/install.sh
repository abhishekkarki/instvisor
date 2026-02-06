#!/bin/bash
set -e

INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/instvisor"
DATA_DIR="/var/lib/instvisor"
SYSTEMD_DIR="/etc/systemd/system"

echo "Installing Instvisor..."

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root (sudo)"
    exit 1
fi

# Create instvisor user
if ! id -u instvisor > /dev/null 2>&1; then
    echo "Creating instvisor user..."
    useradd --system --no-create-home --shell /bin/false instvisor
fi

# Create directories
echo "Creating directories..."
mkdir -p "$CONFIG_DIR" "$DATA_DIR"
chown instvisor:instvisor "$DATA_DIR"
chmod 755 "$DATA_DIR"

# Install binaries
echo "Installing binaries..."
cp build/instvisor-agent "$INSTALL_DIR/"
cp build/instvisor-analyze "$INSTALL_DIR/"
chmod 755 "$INSTALL_DIR/instvisor-agent"
chmod 755 "$INSTALL_DIR/instvisor-analyze"

# Install configuration
if [ ! -f "$CONFIG_DIR/agent.yaml" ]; then
    echo "Installing default configuration..."
    cp configs/agent.yaml "$CONFIG_DIR/"
else
    echo "Configuration already exists, skipping..."
fi

# Install systemd service
if command -v systemctl &> /dev/null; then
    echo "Installing systemd service..."
    cp deployments/systemd/instvisor.service "$SYSTEMD_DIR/"
    systemctl daemon-reload
    
    echo ""
    echo "Installation complete!"
    echo ""
    echo "To start Instvisor:"
    echo "  sudo systemctl start instvisor"
    echo ""
    echo "To enable on boot:"
    echo "  sudo systemctl enable instvisor"
    echo ""
    echo "To view logs:"
    echo "  sudo journalctl -u instvisor -f"
    echo ""
    echo "To analyze metrics:"
    echo "  sudo instvisor-analyze"
else
    echo ""
    echo "Installation complete!"
    echo ""
    echo "Systemd not detected. To run manually:"
    echo "  sudo $INSTALL_DIR/instvisor-agent -config $CONFIG_DIR/agent.yaml"
fi