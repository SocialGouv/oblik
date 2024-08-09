#!/usr/bin/env bash
set -eo errexit

export KUBECONFIG="${HOME}/.kube/config"

$(dirname $0)/kind-with-registry.sh

docker build --tag localhost:5001/oblik:test .
docker push localhost:5001/oblik:test

helm repo add kyverno https://kyverno.github.io/kyverno/
helm repo update
helm upgrade --install kyverno kyverno/kyverno --namespace kyverno --create-namespace --version=3.2.5

./charts/vpa/gencerts.sh
helm template charts/vpa | kubectl apply -f -

helm upgrade --install --create-namespace --namespace oblik \
  --set image.repository=localhost:5001/oblik \
  --set image.tag=test \
  --set env.OBLIK_DEFAULT_MIN_REQUEST_CPU=100m \
  --set env.OBLIK_DEFAULT_MIN_REQUEST_MEMORY=262144k \
  oblik charts/oblik

go test -v ./tests
