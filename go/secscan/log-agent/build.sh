#!/usr/bin/env bash
set -e
docker build --no-cache --platform=linux/amd64 -t saichler/secscan-log-agent:latest .
docker push saichler/secscan-log-agent:latest
