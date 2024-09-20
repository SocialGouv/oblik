#!/usr/bin/env bash
set -eo errexit

export KUBECONFIG="${HOME}/.kube/config"


docker build --tag localhost:5001/oblik:test .
docker push localhost:5001/oblik:test

helm upgrade --install --create-namespace --namespace oblik \
  --set replicas=3 \
  --set image.repository=localhost:5001/oblik \
  --set image.tag=test \
  --set image.pullPolicy=Always \
  --set annotations.refreshtime="$(date +'%F-%H:%m:%S')" \
  --set env.OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_CPU=100m \
  --set env.OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_MEMORY=262144k \
  --set env.OBLIK_DEFAULT_MIN_REQUEST_CPU=100m \
  --set env.OBLIK_DEFAULT_MIN_REQUEST_MEMORY=262144k \
  --set env.OBLIK_DEFAULT_CRON="* * * * *" \
  --set env.OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX="5s" \
  --wait \
  oblik charts/oblik
