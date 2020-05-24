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

function connectivity_tests() {
    local netshoot_pod nginx_svc_ip
    netshoot_pod=$(kubectl get pods -l app=netshoot | awk 'FNR == 2 {print $1}')
    nginx_svc_ip=$(with_context cluster3 get_svc_ip nginx-demo)

    [[ "${lighthouse}" = "true" ]] || return 0
    resolved_ip=$((kubectl exec "${netshoot_pod}" -- ping -c 1 -W 1 nginx-demo 2>/dev/null || :) \
                  | grep PING | awk '{print $3}' | tr -d '()')
    if [[ "$resolved_ip" != "$nginx_svc_ip" ]]; then
        echo "Resolved IP $resolved_ip doesn't match the service ip $nginx_svc_ip"
        exit 1
    fi
}

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

with_context cluster3 deploy_resource "${RESOURCES_DIR}/nginx-demo.yaml"
[[ "${lighthouse}" = "true" ]] && with_context cluster3 kubectl apply -f "${DAPPER_SOURCE}/scripts/kind-e2e/nginx-demo-export.yaml"
with_context cluster2 deploy_resource "${RESOURCES_DIR}/netshoot.yaml"

with_context cluster2 connectivity_tests

# dataplane E2E need to be modified for globalnet
if [[ "${globalnet}" != "true" ]]; then
    # run dataplane E2E tests between the two clusters
    ${DAPPER_SOURCE}/bin/subctl verify-connectivity --verbose \
        ${KUBECONFIGS_DIR}/kind-config-cluster2 \
        ${KUBECONFIGS_DIR}/kind-config-cluster3
fi

print_clusters_message

