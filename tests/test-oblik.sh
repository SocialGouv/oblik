#!/usr/bin/env bash
set -eo errexit
export KUBECONFIG="${HOME}/.kube/config"

go test -count=1 -timeout=30m $@ ./tests