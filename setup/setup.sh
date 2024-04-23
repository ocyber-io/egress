#!/bin/bash

CWD=$(dirname "$(readlink -f "$0")")
GSTREAMER_VERSION=1.22.8
LIBNICE_VERSION=0.1.21
BASE_PATH="$HOME/installation"
mkdir -p $BASE_PATH

sh $CWD/gstreamer/gstreamer-base.sh $GSTREAMER_VERSION $LIBNICE_VERSION $BASE_PATH
sh $CWD/gstreamer/gstreamer-dev.sh $BASE_PATH
sh $CWD/gstreamer/gstreamer-prod.sh $BASE_PATH