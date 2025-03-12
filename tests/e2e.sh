#!/usr/bin/env bash
set -eo errexit
$(dirname $0)/kind-with-registry.sh
$(dirname $0)/install-dependencies.sh
$(dirname $0)/deploy-oblik.sh
$(dirname $0)/test-oblik.sh $@
