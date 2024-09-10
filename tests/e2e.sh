#!/usr/bin/env bash
set -eo errexit
export KUBECONFIG="${HOME}/.kube/config"

# $(dirname $0)/setup-cluster.sh
$(dirname $0)/deploy-oblik.sh
$(dirname $0)/test-oblik.sh
