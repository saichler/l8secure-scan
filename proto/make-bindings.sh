#!/usr/bin/env bash

set -e

# Download proto dependencies for import resolution only -- their Go code
# comes from the vendored l8types/l8common dependencies, not local generation.
wget -q https://raw.githubusercontent.com/saichler/l8types/refs/heads/main/proto/api.proto
wget -q https://raw.githubusercontent.com/saichler/l8common/refs/heads/main/proto/l8common.proto

PROTOS=(
    secscan.proto
)

# Use the protoc image to run protoc.sh and generate Go bindings.
for p in "${PROTOS[@]}"; do
    docker run --user "$(id -u):$(id -g)" -e PROTO="$p" --mount type=bind,source="$PWD",target=/home/proto/ -i saichler/protoc:latest
done

rm -f api.proto l8common.proto

# Move generated bindings to the types directory and clean up
rm -rf ../go/types
mkdir -p ../go/types
mv ./types/* ../go/types/.
rm -rf ./types
rm -rf *.rs

# Fix relative import paths not auto-resolved by the protoc image
cd ../go
find . -name "*.go" -type f -exec sed -i 's|"./types/l8api"|"github.com/saichler/l8types/go/types/l8api"|g' {} +
find . -name "*.go" -type f -exec sed -i 's|"./types/l8common"|"github.com/saichler/l8common/go/types/l8common"|g' {} +

# sed can leave the import block out of gofmt's canonical (path-alphabetical) order
gofmt -w ./types
