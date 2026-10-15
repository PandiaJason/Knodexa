#!/bin/sh
# Run from any directory; Go is the only required programming runtime.
set -eu
cd "$(dirname "$0")/.."
go vet ./...
go test -race ./...
check_dir=$(mktemp -d)
trap 'rm -rf "$check_dir"' EXIT HUP INT TERM
go build -o "$check_dir/knodexa" ./cmd/knodexa
cd "$check_dir"
./knodexa init --name='Standalone smoke test'
./knodexa task create --id=smoke --title='Smoke test' --goal='Validate CLI context' --alias='standalone validation'
./knodexa context smoke --json --budget 500 > context.json
./knodexa resolve 'Continue standalone validation' --json > resolution.json
if ./knodexa context smoke --unknown > /dev/null 2>&1; then
    echo 'Unknown flag was incorrectly accepted' >&2
    exit 1
fi
./knodexa doctor
echo 'Standalone checks passed.'
