#!/usr/bin/env bash

## Process command line flags ##

source /usr/share/shflags/shflags
DEFINE_string 'globalnet' 'false' "Deploy with operlapping CIDRs (set to 'true' to enable)"
DEFINE_string 'lighthouse' 'false' "Deploy with lighthouse"
DEFINE_string 'status' 'onetime' "Status flag (onetime, create, keep, clean)"
FLAGS "$@" || exit $?
eval set -- "${FLAGS_ARGV}"

globalnet="${FLAGS_globalnet}"
lighthouse="${FLAGS_lighthouse}"
status="${FLAGS_status}"
echo "Running with: globalnet=${globalnet}, lighthouse=${lighthouse}, status=${status}"

set -em

source ${SCRIPTS_DIR}/lib/debug_functions
source ${SCRIPTS_DIR}/lib/version
source ${SCRIPTS_DIR}/lib/utils
source ${SCRIPTS_DIR}/lib/deploy_funcs

### Functions ###

function setup_broker() {
    if kubectl --context=cluster1 get crd clusters.submariner.io > /dev/null 2>&1; then
        echo Submariner CRDs already exist, skipping broker creation...
    else
        echo Installing broker on cluster1.
         sd=
         [[ $lighthouse = true ]] && sd=--service-discovery
         gn=
         [[ $globalnet = true ]] && gn=--globalnet
         set -o pipefail
         ${DAPPER_SOURCE}/bin/subctl --kubeconfig ${PRJ_ROOT}/output/kubeconfigs/kind-config-merged --kubecontext cluster1 deploy-broker ${sd} ${gn}|& cat
         set +o pipefail
         [[ $lighthouse = true ]] && kubefedctl federate namespace default --kubefed-namespace kubefed-operator
    fi

    SUBMARINER_BROKER_URL=$(kubectl --context=cluster1 -n default get endpoints kubernetes -o jsonpath="{.subsets[0].addresses[0].ip}:{.subsets[0].ports[?(@.name=='https')].port}")
    SUBMARINER_BROKER_CA=$(kubectl --context=cluster1 -n ${SUBMARINER_BROKER_NS} get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='${SUBMARINER_BROKER_NS}-client')].data['ca\.crt']}")
    SUBMARINER_BROKER_TOKEN=$(kubectl --context=cluster1 -n ${SUBMARINER_BROKER_NS} get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='${SUBMARINER_BROKER_NS}-client')].data.token}"|base64 --decode)
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

function get_globalip() {
    svcname=$1
    context=$2
    gip=
    attempt_counter=0
    max_attempts=30
    # It takes a while for globalIp to show up on a service
    until [[ $gip ]]; do
        if [[ ${attempt_counter} -eq ${max_attempts} ]];then
          echo "Max attempts reached, failed to get globalIp!"
          exit 1
        fi
        sleep 1
        gip=$(kubectl --context=$context get svc $svcname -o jsonpath='{.metadata.annotations.submariner\.io/globalIp}')
        attempt_counter=$(($attempt_counter+1))
    done
    echo $gip
}

function try_connect() {
    target=$1
    netshoot_pod=$(kubectl --context=cluster2 get pods -l app=netshoot | awk 'FNR == 2 {print $1}')

    echo "Testing connectivity between clusters - $netshoot_pod cluster2 --> $target nginx service cluster3"

    attempt_counter=0
    max_attempts=5
    until $(kubectl --context=cluster2 exec ${netshoot_pod} -- curl --output /dev/null -m 30 --silent --head --fail ${target}); do
        if [[ ${attempt_counter} -eq ${max_attempts} ]];then
          echo "Max attempts reached, connection test failed!"
          exit 1
        fi
        attempt_counter=$(($attempt_counter+1))
    done
}

function test_connection() {
    if [[ $globalnet = true ]]; then
        nginx_svc_ip_cluster3=$(get_globalip nginx-demo cluster3)
    else
        nginx_svc_ip_cluster3=$(kubectl --context=cluster3 get svc -l app=nginx-demo | awk 'FNR == 2 {print $3}')
    fi
    if [[ -z "$nginx_svc_ip_cluster3" ]]; then
        echo "Failed to get nginx-demo IP"
        exit 1
    fi
    try_connect $nginx_svc_ip_cluster3
    if [[ $lighthouse = true ]]; then
        resolved_ip=$(kubectl --context=cluster2 exec ${netshoot_pod} -- ping -c 1 -W 1 nginx-demo 2>/dev/null | grep PING | awk '{print $3}')
        # strip the () braces from resolved_ip
        resolved_ip=${resolved_ip:1:-1}
        if [[ "$resolved_ip" != "$nginx_svc_ip_cluster3" ]]; then
            echo "Resolved IP $resolved_ip doesn't match with service ip $nginx_svc_ip_cluster3"
            exit 1
        fi
        try_connect nginx-demo
    fi

    echo "Connection test was successful!"
}

function update_subm_pods() {
    echo Removing submariner engine pods...
    kubectl --context=$1 delete pods -n submariner -l app=submariner-engine
    kubectl --context=$1 wait --for=condition=Ready pods -l app=submariner-engine -n submariner --timeout=60s
    echo Removing submariner route agent pods...
    kubectl --context=$1 delete pods -n submariner -l app=submariner-routeagent
    kubectl --context=$1 wait --for=condition=Ready pods -l app=submariner-routeagent -n submariner --timeout=60s
}

function test_with_e2e_tests {
    cd ${DAPPER_SOURCE}/test/e2e

    go test -args -ginkgo.v -ginkgo.randomizeAllSpecs -ginkgo.reportPassed \
        -dp-context cluster2 -dp-context cluster3  \
        -report-dir ${DAPPER_OUTPUT}/junit 2>&1 | \
        tee ${DAPPER_OUTPUT}/e2e-tests.log
}

function cleanup {
    "${SCRIPTS_DIR}"/cleanup.sh
}

### Main ###

if [[ $status = clean ]]; then
    cleanup
    exit 0
elif [[ $status = onetime ]]; then
    echo Status $status: Will cleanup on EXIT signal
    trap cleanup EXIT
elif [[ $status != keep && $status != create ]]; then
    echo Unknown status: $status
    cleanup
    exit 1
fi

PRJ_ROOT=$(git rev-parse --show-toplevel)
SUBMARINER_BROKER_NS=submariner-k8s-broker
# FIXME: This can change and break re-running deployments
SUBMARINER_PSK=$(cat /dev/urandom | LC_CTYPE=C tr -dc 'a-zA-Z0-9' | fold -w 64 | head -n 1)
declare_kubeconfig

kubectl config view --flatten > ${PRJ_ROOT}/output/kubeconfigs/kind-config-merged

kind_import_images
setup_broker

context=cluster1
kubectl config use-context $context

# Import functions for testing with Operator
# NB: These are also used to verify non-Operator deployments, thereby asserting the two are mostly equivalent
. ${DAPPER_SOURCE}/scripts/kind-e2e/lib_operator_verify_subm.sh

create_subm_vars
verify_subm_broker_secrets

. ${DAPPER_SOURCE}/scripts/kind-e2e/lib_operator_deploy_subm.sh

for i in 2 3; do
    context=cluster$i
    kubectl config use-context $context

    # Add SubM gateway labels
    add_subm_gateway_label
    # Verify SubM gateway labels
    verify_subm_gateway_label

    # Deploy SubM Operator
    if [ "${context}" = "cluster2" ] || [ "${context}" = "cluster3" ]; then
        set -o pipefail
        ${DAPPER_SOURCE}/bin/subctl join --operator-image "${subm_engine_image_repo}/submariner-operator:local" \
                        --kubeconfig ${PRJ_ROOT}/output/kubeconfigs/kind-config-merged \
                        --kubecontext ${context} \
                        --clusterid ${context} \
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
    else
        echo Unknown context ${context}
        exit 1
    fi

    # Verify shared CRDs
    verify_endpoints_crd
    verify_clusters_crd

    # Verify SubM CRD
    verify_subm_crd
    # Verify SubM Operator
    verify_subm_operator
    # Verify SubM Operator pod
    verify_subm_op_pod
    # Verify SubM Operator container
    verify_subm_operator_container

    # FIXME: Rename all of these submariner-engine or engine, vs submariner
    # Verify SubM CR
    verify_subm_cr
    # Verify SubM Engine Deployment
    verify_subm_engine_deployment
    # Verify SubM Engine Pod
    verify_subm_engine_pod
    # Verify SubM Engine container
    verify_subm_engine_container
    # Verify Engine secrets
    verify_subm_engine_secrets

    # Verify SubM Routeagent DaemonSet
    verify_subm_routeagent_daemonset
    # Verify SubM Routeagent Pods
    verify_subm_routeagent_pod
    # Verify SubM Routeagent container
    verify_subm_routeagent_container

    if [[ $globalnet = true ]]; then
        #Verify SubM Globalnet Daemonset
        verify_subm_globalnet_daemonset
    fi
done

echo "Running subctl a second time to verify if running subctl a second time works fine"

set -o pipefail
${DAPPER_SOURCE}/bin/subctl join --operator-image "${subm_engine_image_repo}/submariner-operator:local" \
                --kubeconfig ${PRJ_ROOT}/output/kubeconfigs/kind-config-merged \
                --kubecontext ${context} \
                --clusterid ${context} \
                --repository "${subm_engine_image_repo}" \
                --version ${subm_engine_image_tag} \
                --nattport ${ce_ipsec_nattport} \
                --ikeport ${ce_ipsec_ikeport} \
                --colorcodes ${subm_colorcodes} \
                --broker-cluster-context cluster1 \
                --disable-nat broker-info.subm |& cat
set +o pipefail

deploy_netshoot_cluster2
deploy_nginx_cluster3

test_connection

# dataplane E2E need to be modified for globalnet
if [[ $globalnet = false ]]; then
    # run dataplane E2e tests between the two clusters
    ${DAPPER_SOURCE}/bin/subctl verify-connectivity ${PRJ_ROOT}/output/kubeconfigs/kind-config-cluster2 \
                                      ${PRJ_ROOT}/output/kubeconfigs/kind-config-cluster3 \
                                      --verbose
fi

if [[ ${status} = keep ]]; then
    echo "your 3 virtual clusters are deployed and working properly with your local"
    echo "submariner source code, and can be accessed with:"
    echo ""
    echo "export KUBECONFIG=\$(echo \$(git rev-parse --show-toplevel)/output/kubeconfigs/kind-config-cluster{1..3} | sed 's/ /:/g')"
    echo ""
    echo "$ kubectl config use-context cluster1 # or cluster2, cluster3.."
    echo ""
    echo "to cleanup, just run: make e2e status=clean"
fi
