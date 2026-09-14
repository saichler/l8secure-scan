#!/usr/bin/env bash
set -e
docker build --no-cache --platform=linux/amd64 -t saichler/secscan-scanner:latest .
docker push saichler/secscan-scanner:latest
