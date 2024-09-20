#!/usr/bin/env bash
set -eo errexit
./charts/vpa/gencerts.sh
helm template charts/vpa | kubectl apply -f -
