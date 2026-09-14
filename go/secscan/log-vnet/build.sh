#!/usr/bin/env bash
set -e
docker build --no-cache --platform=linux/amd64 -t saichler/secscan-log-vnet:latest .
docker push saichler/secscan-log-vnet:latest
