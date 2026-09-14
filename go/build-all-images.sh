#!/usr/bin/env bash
set -e
cd secscan
echo "*** Building Vnet ***"
cd ./vnet
if ! ./build.sh; then echo "FAILED to build Vnet"; exit 1; fi
echo "*** Building Backend ***"
cd ../main
if ! ./build.sh; then echo "FAILED to build Backend"; exit 1; fi
echo "*** Building Scanner ***"
cd ../scanner
if ! ./build.sh; then echo "FAILED to build Scanner"; exit 1; fi
echo "*** Building Web ***"
cd ../ui
if ! ./build.sh; then echo "FAILED to build Web"; exit 1; fi
echo "*** Building Log Vnet ***"
cd ../log-vnet
if ! ./build.sh; then echo "FAILED to build Log Vnet"; exit 1; fi
echo "*** Building Log Agent ***"
cd ../log-agent
if ! ./build.sh; then echo "FAILED to build Log Agent"; exit 1; fi
echo "*** All secscan images built ***"
