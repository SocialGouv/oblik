#!/usr/bin/env bash
set -eo errexit

# build

docker build --tag localhost:5001/oblik:test .
docker push localhost:5001/oblik:test


# deploy
export OBLIK_TEST_DISABLE_HA=${OBLIK_TEST_DISABLE_HA:-""}
if [ -n "$OBLIK_TEST_DISABLE_HA" ]; then
  REPLICAS=1
else
  REPLICAS=3
fi

helm upgrade --install --create-namespace --namespace oblik \
  --set replicas=$REPLICAS \
  --set image.repository=localhost:5001/oblik \
  --set image.tag=test \
  --set image.pullPolicy=Always \
  --set annotations.refreshtime="$(date +'%F-%H:%m:%S')" \
  --set env.OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_CPU=25m \
  --set env.OBLIK_DEFAULT_UNPROVIDED_APPLY_DEFAULT_REQUEST_MEMORY=250Mi \
  --set env.OBLIK_DEFAULT_MIN_REQUEST_CPU=100m \
  --set env.OBLIK_DEFAULT_MIN_REQUEST_MEMORY=250Mi \
  --set env.OBLIK_DEFAULT_CRON="* * * * *" \
  --set env.OBLIK_DEFAULT_CRON_ADD_RANDOM_MAX="5s" \
  --set resources.requests.cpu=100m \
  --set resources.requests.memory=256Mi \
  --set resources.limits.cpu=500m \
  --set resources.limits.memory=512Mi \
  --wait \
  oblik charts/oblik
