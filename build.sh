#!/bin/sh
set -ex

CGO_ENABLED=0 go build -ldflags "-s -w" -o itsy
sudo setcap CAP_NET_BIND_SERVICE=+eip ./itsy