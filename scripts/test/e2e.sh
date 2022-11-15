#!/usr/bin/env bash

. "${SCRIPTS_DIR}/lib/debug_functions"
. "${SCRIPTS_DIR}/lib/utils"
. "${SCRIPTS_DIR}/lib/kubecfg"

set -em -o pipefail

subm_ns=submariner-operator

### Main ###

# Run project specific E2E tests (they don't overlap with the generic ones)
# This will deploy the environment if necessary
"${SCRIPTS_DIR}"/e2e.sh "$@"

# Reload KUBECONFIG in case deploy was triggered
. "${SCRIPTS_DIR}/lib/kubecfg"

# Check that we're testing the version we expect
function check_operator_version() {
    local opv
    opv=$(kubectl --context="$cluster" logs -n submariner-operator -l name=submariner-operator --tail=-1 | awk '/Submariner operator version/ { print $NF }')
    printf "Cluster %s is running operator version %s\n" "$cluster" "$opv"
    if [ "$opv" != "$VERSION" ]; then
        printf "Operator version mismatch: expected %s, got %s\n" "$VERSION" "$opv"
        exit 1
    fi
}
run_consecutive "$*" check_operator_version

command -v subctl || curl -Ls https://get.submariner.io | VERSION=devel bash

load_settings
verify="connectivity"
[[ "${LIGHTHOUSE}" != "true" ]] || verify="service-discovery"

# Run generic E2E tests between the clusters
subctl verify --only "${verify}" --submariner-namespace="$subm_ns" \
    --verbose --connection-timeout 20 --connection-attempts 4 \
    --context "${clusters[0]}" --tocontext "${clusters[1]}"
