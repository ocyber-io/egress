#!/bin/bash

CWD=$(dirname "$(readlink -f "$0")")
BASE_PATH=$1

# Set environment variables
export DEBUG=false
export OPTIMIZATIONS=true
export PATH=/root/.cargo/bin:$PATH

# Copy gst-plugins-rs from the development stage
#cp -r ./gst-plugins-rs ./gst-plugins-rs

# Copy compile-rs script
chmod +x $CWD/compile-rs

# Run compile-rs script
$CWD/compile-rs $BASE_PATH

# Copy compiled binaries from the development stage to the production stage
#cp -r /compiled-binaries /

