#!/usr/bin/env bash

set -em -o pipefail

source ${SCRIPTS_DIR}/lib/debug_functions
source ${SCRIPTS_DIR}/lib/utils
source ${SCRIPTS_DIR}/lib/deploy_funcs
source ${SCRIPTS_DIR}/lib/deploy_operator
source ${SCRIPTS_DIR}/lib/cluster_settings
source ${DAPPER_SOURCE}/scripts/kind-e2e/cluster_settings

### Variables ###

[[ ! ${DEPLOY_ARGS} =~ "--globalnet" ]] || globalnet=true
[[ ! ${DEPLOY_ARGS} =~ "--service-discovery" ]] || lighthouse=true
timeout="2m" # Used by deploy_resource

### Functions ###

### Main ###

declare_kubeconfig

# Import functions for testing with Operator
# NB: These are also used to verify non-Operator deployments, thereby asserting the two are mostly equivalent
. ${DAPPER_SOURCE}/scripts/kind-e2e/lib_operator_verify_subm.sh

create_subm_vars
with_context "$broker" broker_vars

with_context "$broker" verify_subm_broker_secrets

run_subm_clusters verify_subm_deployed

echo "Running subctl a second time to verify if running subctl a second time works fine"
with_context cluster3 subctl_install_subm

verify="--connectivity"
if [[ "$lighthouse" == "true" ]]; then
    verify="--all"
fi
echo "Verify is set to: $verify"

# run dataplane E2E tests between the two clusters
${DAPPER_SOURCE}/bin/subctl verify ${verify} --verbose \
    ${KUBECONFIGS_DIR}/kind-config-cluster2 \
    ${KUBECONFIGS_DIR}/kind-config-cluster3

print_clusters_message

