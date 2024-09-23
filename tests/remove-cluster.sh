#!/usr/bin/env bash
set -eo errexit

export KUBECONFIG="${HOME}/.kube/config"

kind delete cluster
docker rm -f kind-registry

