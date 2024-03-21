#!/bin/bash

CWD=$(dirname "$(readlink -f "$0")")
BASE_PATH=$1

# Set environment variables
export DEBUG=true
export OPTIMIZATIONS=false

# Copy compile scripts
chmod +x $CWD/compile
chmod +x $CWD/compile-rs

# Run compile scripts
$CWD/compile $BASE_PATH
$CWD/compile-rs $BASE_PATH

# Install dependencies
chmod +x $CWD/install-dependencies
$CWD/install-dependencies $BASE_PATH

# Copy compiled binaries from the previous stage
#cp -r ./compiled-binaries/* /

cd $CWD
