#!/bin/bash
# spawnd installer - Downloads from S3 for fast regional access
set -e

echo "=== Installing spawnd ==="

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)
        BINARY="spawnd-linux-amd64"
        ;;
    aarch64|arm64)
        BINARY="spawnd-linux-arm64"
        ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

echo "Architecture: $ARCH ($BINARY)"

# Detect region from instance metadata
echo "Detecting AWS region..."
TOKEN=$(curl -X PUT "http://169.254.169.254/latest/api/token" -H "X-aws-ec2-metadata-token-ttl-seconds: 21600" 2>/dev/null)
if [ -n "$TOKEN" ]; then
    REGION=$(curl -H "X-aws-ec2-metadata-token: $TOKEN" http://169.254.169.254/latest/meta-data/placement/region 2>/dev/null)
else
    # Fallback without token
    REGION=$(curl -s http://169.254.169.254/latest/meta-data/placement/region 2>/dev/null)
fi

if [ -z "$REGION" ]; then
    echo "⚠️  Could not detect region, using us-east-1"
    REGION="us-east-1"
else
    echo "Region: $REGION"
fi

# S3 bucket name (regional)
S3_BUCKET="spawn-binaries-${REGION}"
S3_PATH="s3://${S3_BUCKET}/${BINARY}"

echo "Downloading spawnd from ${S3_PATH}..."

# Try regional bucket first
if aws s3 cp "$S3_PATH" /usr/local/bin/spawnd --region "$REGION" 2>/dev/null; then
    echo "✅ Downloaded from regional bucket"
else
    echo "⚠️  Regional bucket not available, trying us-east-1..."
    # Fallback to us-east-1
    aws s3 cp "s3://spawn-binaries-us-east-1/${BINARY}" /usr/local/bin/spawnd --region us-east-1
    if [ $? -eq 0 ]; then
        echo "✅ Downloaded from us-east-1"
    else
        echo "❌ Failed to download spawnd"
        exit 1
    fi
fi

# Make executable
chmod +x /usr/local/bin/spawnd

# Verify installation
if /usr/local/bin/spawnd version; then
    echo "✅ spawnd installed successfully"
else
    echo "❌ spawnd binary verification failed"
    exit 1
fi

# Create systemd service
echo "Installing systemd service..."
cat > /etc/systemd/system/spawnd.service <<'EOF'
[Unit]
Description=Spawn Agent - Instance self-monitoring
Documentation=https://github.com/yourusername/spawn
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/spawnd
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

# Security hardening
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
systemctl daemon-reload

# Enable and start spawnd
systemctl enable spawnd
systemctl start spawnd

# Wait a moment for startup
sleep 2

# Check status
if systemctl is-active --quiet spawnd; then
    echo "✅ spawnd is running"
    systemctl status spawnd --no-pager --lines=5
else
    echo "⚠️  spawnd may have issues"
    journalctl -u spawnd -n 20 --no-pager
fi

echo ""
echo "=== Installation complete ==="
echo "View logs: journalctl -u spawnd -f"
echo "Check status: systemctl status spawnd"

