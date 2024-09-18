#!/usr/bin/env bash
set -eo errexit
export KUBECONFIG="${HOME}/.kube/config"

# DEBUG=true go test -v -count=1 ./tests
go test -count=1 $@ ./tests