#!/usr/bin/env bash
set -e
docker build --no-cache --platform=linux/amd64 -t saichler/secscan:latest .
docker push saichler/secscan:latest
