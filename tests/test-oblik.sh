#!/usr/bin/env bash
set -eo errexit
go test -count=1 -timeout=30m -v ./tests $@