#!/bin/sh
# Runs all unit tests of the repository.
# Usage: ./test.sh [package ...]
#   Without arguments, tests every package.
#   With arguments, only the given packages are tested, e.g.:
#   ./test.sh ./subtitle/ ./utils/
set -e

cd "$(dirname "$0")"

if [ $# -gt 0 ]; then
    go test -v "$@"
else
    go test ./...
fi
