#!/bin/bash

# Define arguments
CWD=$(dirname "$(readlink -f "$0")")
GSTREAMER_VERSION=$1
LIBNICE_VERSION=$2
BASE_PATH=$3

# Install dependencies
chmod +x $CWD/install-dependencies
$CWD/install-dependencies $BASE_PATH


# Set environment variables
export PATH=/root/.cargo/bin:$PATH

cd $BASE_PATH

# Download and extract GStreamer libraries
for lib in gstreamer gst-plugins-base gst-plugins-good gst-plugins-bad gst-plugins-ugly gst-libav; do
    wget https://gstreamer.freedesktop.org/src/$lib/$lib-$GSTREAMER_VERSION.tar.xz
    tar -xf $lib-$GSTREAMER_VERSION.tar.xz
    rm $lib-$GSTREAMER_VERSION.tar.xz
    mv $lib-$GSTREAMER_VERSION $lib
done

# Download and extract Rust plugins from GitLab
wget https://gitlab.freedesktop.org/gstreamer/gst-plugins-rs/-/archive/gstreamer-$GSTREAMER_VERSION/gst-plugins-rs-gstreamer-$GSTREAMER_VERSION.tar.gz
tar xfz gst-plugins-rs-gstreamer-$GSTREAMER_VERSION.tar.gz
rm gst-plugins-rs-gstreamer-$GSTREAMER_VERSION.tar.gz
mv gst-plugins-rs-gstreamer-$GSTREAMER_VERSION gst-plugins-rs

# Download and extract libnice
wget https://libnice.freedesktop.org/releases/libnice-$LIBNICE_VERSION.tar.gz
tar xfz libnice-$LIBNICE_VERSION.tar.gz
rm libnice-$LIBNICE_VERSION.tar.gz
mv libnice-$LIBNICE_VERSION libnice

cd $CWD
