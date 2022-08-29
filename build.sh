#!/bin/sh
set -ex

CGO_ENABLED=0 go build
sudo setcap CAP_NET_BIND_SERVICE=+eip ./itsy-bitsy-teenie-weenie-port-forwarder-programini