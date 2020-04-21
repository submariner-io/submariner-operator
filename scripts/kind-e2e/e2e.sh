#!/usr/bin/env bash

## Process command line flags ##

source /usr/share/shflags/shflags
DEFINE_string 'globalnet' 'false' "Deploy with operlapping CIDRs (set to 'true' to enable)"
DEFINE_string 'lighthouse' 'false' "Deploy with lighthouse"
FLAGS "$@" || exit $?
eval set -- "${FLAGS_ARGV}"

globalnet="${FLAGS_globalnet}"
lighthouse="${FLAGS_lighthouse}"
echo "Running with: globalnet=${globalnet}, lighthouse=${lighthouse}"

set -em

source ${SCRIPTS_DIR}/lib/debug_functions
source ${SCRIPTS_DIR}/lib/version
source ${SCRIPTS_DIR}/lib/utils
source ${SCRIPTS_DIR}/lib/deploy_funcs
source ${DAPPER_SOURCE}/scripts/kind-e2e/cluster_settings

### Functions ###

function setup_broker() {
    if kubectl get crd clusters.submariner.io > /dev/null 2>&1; then
        echo Submariner CRDs already exist, skipping broker creation...
        return
    fi

    echo Installing broker on ${cluster}.
    local sd gn
    [[ $lighthouse = true ]] && sd=--service-discovery
    [[ $globalnet = true ]] && gn=--globalnet
    set -o pipefail
    ${DAPPER_SOURCE}/bin/subctl --kubeconfig ${PRJ_ROOT}/output/kubeconfigs/kind-config-merged --kubecontext ${cluster} deploy-broker ${sd} ${gn}|& cat
    set +o pipefail
    [[ $lighthouse != true ]] || kubefedctl federate namespace default --kubefed-namespace kubefed-operator
}

function broker_vars() {
    SUBMARINER_BROKER_URL=$(kubectl -n default get endpoints kubernetes -o jsonpath="{.subsets[0].addresses[0].ip}:{.subsets[0].ports[?(@.name=='https')].port}")
    SUBMARINER_BROKER_CA=$(kubectl -n ${SUBMARINER_BROKER_NS} get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='${SUBMARINER_BROKER_NS}-client')].data['ca\.crt']}")
    SUBMARINER_BROKER_TOKEN=$(kubectl -n ${SUBMARINER_BROKER_NS} get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='${SUBMARINER_BROKER_NS}-client')].data.token}"|base64 --decode)
}

function kind_import_images() {
    import_image quay.io/submariner/submariner
    import_image quay.io/submariner/submariner-route-agent
    import_image quay.io/submariner/submariner-operator
    [[ $globalnet != "true" ]] || import_image quay.io/submariner/submariner-globalnet
}

function create_subm_vars() {
    # FIXME A better name might be submariner-engine, but just kinda-matching submariner-<random hash> name used by Helm/upstream tests
    deployment_name=submariner
    operator_deployment_name=submariner-operator
    engine_deployment_name=submariner-engine
    routeagent_deployment_name=submariner-routeagent
    broker_deployment_name=submariner-k8s-broker
    globalnet_deployment_name=submariner-globalnet

    declare_cidrs
    natEnabled=false

    subm_engine_image_repo="localhost:5000"
    subm_engine_image_tag=local

    # FIXME: Actually act on this size request in controller
    subm_engine_size=3
    subm_colorcodes=blue
    subm_debug=false
    subm_broker=k8s
    subm_cabledriver=strongswan
    ce_ipsec_debug=false
    ce_ipsec_ikeport=500
    ce_ipsec_nattport=4500

    subm_ns=submariner-operator
    subm_broker_ns=submariner-k8s-broker
}

function deploy_subm() {
    # Add SubM gateway labels
    add_subm_gateway_label
    # Verify SubM gateway labels
    verify_subm_gateway_label

    set -o pipefail
    ${DAPPER_SOURCE}/bin/subctl join --operator-image "${subm_engine_image_repo}/submariner-operator:local" \
                    --kubeconfig ${PRJ_ROOT}/output/kubeconfigs/kind-config-merged \
                    --kubecontext ${cluster} \
                    --clusterid ${cluster} \
                    --repository "${subm_engine_image_repo}" \
                    --version ${subm_engine_image_tag} \
                    --nattport ${ce_ipsec_nattport} \
                    --ikeport ${ce_ipsec_ikeport} \
                    --colorcodes ${subm_colorcodes} \
                    --cable-driver ${subm_cabledriver} \
                    --broker-cluster-context "cluster1" \
                    --disable-nat \
                    broker-info.subm |& cat
    set +o pipefail
}

function connectivity_tests() {
    local netshoot_pod nginx_svc_ip
    netshoot_pod=$(kubectl get pods -l app=netshoot | awk 'FNR == 2 {print $1}')
    nginx_svc_ip=$(with_context cluster3 get_svc_ip nginx-demo)

    with_retries 5 test_connection "$netshoot_pod" "$nginx_svc_ip"
    if [[ $lighthouse = true ]]; then
        resolved_ip=$(kubectl exec "${netshoot_pod}" -- ping -c 1 -W 1 nginx-demo 2>/dev/null \
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

PRJ_ROOT=$(git rev-parse --show-toplevel)
SUBMARINER_BROKER_NS=submariner-k8s-broker
# FIXME: This can change and break re-running deployments
SUBMARINER_PSK=$(cat /dev/urandom | LC_CTYPE=C tr -dc 'a-zA-Z0-9' | fold -w 64 | head -n 1)
declare_kubeconfig

kubectl config view --flatten > ${PRJ_ROOT}/output/kubeconfigs/kind-config-merged

kind_import_images
with_context cluster1 setup_broker
with_context cluster1 broker_vars

# Import functions for testing with Operator
# NB: These are also used to verify non-Operator deployments, thereby asserting the two are mostly equivalent
. ${DAPPER_SOURCE}/scripts/kind-e2e/lib_operator_verify_subm.sh

create_subm_vars
with_context cluster1 verify_subm_broker_secrets

if [[ $globalnet = "true" ]]; then
    run_sequential "2 3" deploy_subm
else
    run_parallel "2 3" deploy_subm
fi

run_parallel "2 3" verify_subm_deployed

echo "Running subctl a second time to verify if running subctl a second time works fine"
with_context cluster3 deploy_subm

with_context cluster2 deploy_resource "${RESOURCES_DIR}/netshoot.yaml"
with_context cluster3 deploy_resource "${RESOURCES_DIR}/nginx-demo.yaml"

with_context cluster2 connectivity_tests

# dataplane E2E need to be modified for globalnet
if [[ $globalnet = false ]]; then
    # run dataplane E2e tests between the two clusters
    ${DAPPER_SOURCE}/bin/subctl verify-connectivity ${PRJ_ROOT}/output/kubeconfigs/kind-config-cluster2 \
                                      ${PRJ_ROOT}/output/kubeconfigs/kind-config-cluster3 \
                                      --verbose
fi

cat << EOM
Your 3 virtual clusters are deployed and working properly with your local submariner source code, and can be accessed with:

export KUBECONFIG=\$(echo \$(git rev-parse --show-toplevel)/output/kubeconfigs/kind-config-cluster{1..3} | sed 's/ /:/g')

$ kubectl config use-context cluster1 # or cluster2, cluster3..

To clean evertyhing up, just run: make cleanup
EOM
