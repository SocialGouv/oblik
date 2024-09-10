#!/usr/bin/env bash
set -eo errexit
export KUBECONFIG="${HOME}/.kube/config"

go test -v -count=1 ./tests