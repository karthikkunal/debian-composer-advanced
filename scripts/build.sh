#!/bin/bash
set -e
# Build script for debian-composer-go
go build -o bin/debian-composer-go ./cmd/debian-composer
echo "Build complete."
