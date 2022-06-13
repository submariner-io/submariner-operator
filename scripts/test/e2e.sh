#!/usr/bin/env bash

. "${SCRIPTS_DIR}/lib/debug_functions"
. "${SCRIPTS_DIR}/lib/utils"
. "${SCRIPTS_DIR}/lib/kubecfg"

set -em -o pipefail

subm_ns=submariner-operator
settings="${SETTINGS}"

### Main ###

load_settings
verify="connectivity"
[[ ! ${DEPLOY_ARGS} =~ "service-discovery" ]] || verify="service-discovery"
contexts="${clusters[@]}"

# Run generic E2E tests between the clusters
subctl verify --only "${verify}" --submariner-namespace="$subm_ns" \
    --verbose --connection-timeout 20 --connection-attempts 4 \
    --kubecontexts "${contexts//${IFS:0:1}/,}"

# Run project specific E2E tests (they don't overlap with the generic ones)
${SCRIPTS_DIR}/e2e.sh "$@"
