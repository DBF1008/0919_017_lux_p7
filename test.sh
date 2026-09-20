#!/bin/sh
# Run all unit tests, mirroring the Test step of .github/workflows/ci.yml.
set -e

go test -race -coverpkg=./... -coverprofile=coverage.txt ./...
