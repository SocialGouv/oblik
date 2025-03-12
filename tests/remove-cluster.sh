#!/usr/bin/env bash
set -eo errexit

kind delete cluster
docker rm -f kind-registry

