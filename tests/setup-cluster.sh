#!/usr/bin/env bash
set -eo errexit

export KUBECONFIG="${HOME}/.kube/config"

$(dirname $0)/kind-with-registry.sh

helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update
helm upgrade --install kyverno kyverno/kyverno --namespace kyverno --create-namespace --version=3.2.5

./charts/vpa/gencerts.sh
helm template charts/vpa | kubectl apply -f -


