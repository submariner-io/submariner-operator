#!/usr/bin/env bash

set -em -o pipefail

source ${SCRIPTS_DIR}/lib/debug_functions
source ${SCRIPTS_DIR}/lib/utils
source ${SCRIPTS_DIR}/lib/deploy_funcs
source ${SCRIPTS_DIR}/lib/deploy_operator
source ${DAPPER_SOURCE}/scripts/kind-e2e/cluster_settings

### Variables ###

[[ ! ${DEPLOY_ARGS} =~ "--globalnet" ]] || globalnet=true
[[ ! ${DEPLOY_ARGS} =~ "--service-discovery" ]] || lighthouse=true

### Functions ###

function connectivity_tests() {
    local netshoot_pod nginx_svc_ip
    netshoot_pod=$(kubectl get pods -l app=netshoot | awk 'FNR == 2 {print $1}')
    nginx_svc_ip=$(with_context cluster3 get_svc_ip nginx-demo)

    if [[ $lighthouse = true ]]; then
        resolved_ip=$((kubectl exec "${netshoot_pod}" -- ping -c 1 -W 1 nginx-demo 2>/dev/null || :) \
                      | grep PING | awk '{print $3}' | tr -d '()')
        if [[ "$resolved_ip" != "$nginx_svc_ip" ]]; then
            echo "Resolved IP $resolved_ip doesn't match the service ip $nginx_svc_ip"
            exit 1
        fi

        with_retries 5 test_connection "$netshoot_pod" nginx-demo
    fi
}

function test_with_e2e_tests {
    cd ${DAPPER_SOURCE}/test/e2e

    go test -args -ginkgo.v -ginkgo.randomizeAllSpecs -ginkgo.reportPassed \
        -dp-context cluster2 -dp-context cluster3  \
        -report-dir ${DAPPER_OUTPUT}/junit 2>&1 | \
        tee ${DAPPER_OUTPUT}/e2e-tests.log
}

### Main ###

declare_kubeconfig

# Import functions for testing with Operator
# NB: These are also used to verify non-Operator deployments, thereby asserting the two are mostly equivalent
. ${DAPPER_SOURCE}/scripts/kind-e2e/lib_operator_verify_subm.sh

create_subm_vars
with_context cluster1 broker_vars

with_context cluster1 verify_subm_broker_secrets

run_parallel "2 3" verify_subm_deployed

echo "Running subctl a second time to verify if running subctl a second time works fine"
with_context cluster3 subctl_install_subm

with_context cluster2 deploy_resource "${RESOURCES_DIR}/netshoot.yaml"
with_context cluster3 deploy_resource "${RESOURCES_DIR}/nginx-demo.yaml"

with_context cluster2 connectivity_tests

# dataplane E2E need to be modified for globalnet
if [[ $globalnet != true ]]; then
    # run dataplane E2e tests between the two clusters
    ${DAPPER_SOURCE}/bin/subctl verify-connectivity ${DAPPER_OUTPUT}/kubeconfigs/kind-config-cluster2 \
                                      ${DAPPER_OUTPUT}/kubeconfigs/kind-config-cluster3 \
                                      --verbose
fi

cat << EOM
Your 3 virtual clusters are deployed and working properly with your local source code, and can be accessed with:

export KUBECONFIG=\$(echo \$(git rev-parse --show-toplevel)/output/kubeconfigs/kind-config-cluster{1..3} | sed 's/ /:/g')

$ kubectl config use-context cluster1 # or cluster2, cluster3..

To clean evertyhing up, just run: make cleanup
EOM

