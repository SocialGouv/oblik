#!/usr/bin/env bash
set -eo errexit

helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update
helm upgrade --install kyverno kyverno/kyverno --namespace kyverno --create-namespace --version=3.2.5 \
    --set replicaCount=1 \
    --set resources.limits.cpu=100m \
    --set resources.limits.memory=128Mi \
    --set resourceFilters.enabled=false \
    --set webhooksCleanup.enabled=false \
    --set config.metricsConfig.enabled=false \
    --set config.webhookTimeoutSeconds=5 \
    --set waitServer.enabled=false

./charts/vpa/gencerts.sh
helm template charts/vpa | kubectl apply -f -
