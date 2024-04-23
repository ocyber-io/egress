#!/bin/bash

GOARCH=amd64

wget https://go.dev/dl/go1.21.6.linux-${GOARCH}.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go1.21.6.linux-${GOARCH}.tar.gz
export PATH="/usr/local/go/bin:${PATH}"

# Download go modules
cp go.mod .
cp go.sum .
go mod download

# Copy source
cp -r cmd/ pkg/ version/ /workspace/

# Copy templates
cp -r workspace/build/egress-templates/* cmd/server/templates/
find cmd/server/templates/ -name '*.map' -delete

# Build
# Install dependencies
apt-get update && \
    apt-get install -y \
    curl \
    fonts-noto \
    gnupg \
    pulseaudio \
    unzip \
    wget \
    xvfb \
    gstreamer1.0-plugins-base-

# Install Chrome
cp -r /chrome-installer /chrome-installer
/chrome-installer/install-chrome "$TARGETPLATFORM"

# Clean up
rm -rf /var/lib/apt/lists/*

# Create egress user
useradd -ms /bin/bash -g root -G sudo,pulse,pulse-access egress
mkdir -p /home/egress/tmp /home/egress/.cache/xdgr
chown -R egress /home/egress

# Copy files
cp /workspace/egress /bin/
cp /entrypoint.sh /

# Set environment variables
export PATH=${PATH}:/chrome
export XDG_RUNTIME_DIR=/home/egress/.cache/xdgr
export CHROME_DEVEL_SANDBOX=/usr/local/sbin/chrome-devel-sandbox

# Run
su - egress -c '/entrypoint.sh'
