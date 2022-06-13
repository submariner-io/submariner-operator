#!/usr/bin/env bash

. "${SCRIPTS_DIR}/lib/debug_functions"
. "${SCRIPTS_DIR}/lib/utils"
. "${SCRIPTS_DIR}/lib/kubecfg"

set -em -o pipefail

subm_ns=submariner-operator
settings="${SETTINGS}"

### Main ###

# Run project specific E2E tests (they don't overlap with the generic ones)
# This will deploy the environment if necessary
${SCRIPTS_DIR}/e2e.sh "$@"

# Reload KUBECONFIG in case deploy was triggered
. "${SCRIPTS_DIR}/lib/kubecfg"

command -v subctl || curl -Ls https://get.submariner.io | VERSION=devel bash

load_settings
verify="connectivity"
[[ ! ${DEPLOY_ARGS} =~ "service-discovery" ]] || verify="service-discovery"
contexts="${clusters[@]}"

# Run generic E2E tests between the clusters
subctl verify --only "${verify}" --submariner-namespace="$subm_ns" \
    --verbose --connection-timeout 20 --connection-attempts 4 \
    --kubecontexts "${contexts//${IFS:0:1}/,}"
