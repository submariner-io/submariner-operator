#!/usr/bin/env bash
set -em

source $(git rev-parse --show-toplevel)/scripts/lib/debug_functions

### Functions ###

function kind_clusters() {
    status=$1
    version=$2
    pids=(-1 -1 -1)
    logs=()
    for i in 1 2 3; do
        if [[ $(kind get clusters | grep cluster${i} | wc -l) -gt 0  ]]; then
            echo Cluster cluster${i} already exists, skipping cluster creation...
        else
            logs[$i]=$(mktemp)
            echo Creating cluster${i}, logging to ${logs[$i]}...
            (
            if [[ -n ${version} ]]; then
                kind create cluster --image=kindest/node:v${version} --name=cluster${i} --config=${PRJ_ROOT}/scripts/kind-e2e/cluster${i}-config.yaml
            else
                kind create cluster --name=cluster${i} --config=${PRJ_ROOT}/scripts/kind-e2e/cluster${i}-config.yaml
            fi
            master_ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' cluster${i}-control-plane | head -n 1)
            sed -i -- "s/user: kubernetes-admin/user: cluster$i/g" $(kind get kubeconfig-path --name="cluster$i")
            sed -i -- "s/name: kubernetes-admin.*/name: cluster$i/g" $(kind get kubeconfig-path --name="cluster$i")
            sed -i -- "s/current-context: kubernetes-admin.*/current-context: cluster$i/g" $(kind get kubeconfig-path --name="cluster$i")

            if [[ ${status} = keep ]]; then
                cp -r $(kind get kubeconfig-path --name="cluster$i") ${PRJ_ROOT}/output/kind-config/local-dev/kind-config-cluster${i}
            fi

            sed -i -- "s/server: .*/server: https:\/\/$master_ip:6443/g" $(kind get kubeconfig-path --name="cluster$i")
            cp -r $(kind get kubeconfig-path --name="cluster$i") ${PRJ_ROOT}/output/kind-config/dapper/kind-config-cluster${i}
            ) > ${logs[$i]} 2>&1 &
            set pids[$i] = $!
        fi
    done
    if [[ ${#logs[@]} -gt 0 ]]; then
        echo "(Watch the installation processes with \"tail -f ${logs[*]}\".)"
        for i in 1 2 3; do
            if [[ pids[$i] -gt -1 ]]; then
                wait ${pids[$i]}
                if [[ $? -ne 0 && $? -ne 127 ]]; then
                    echo Cluster $i creation failed:
                    cat ${logs[$i]}
                fi
                rm -f ${logs[$i]}
            fi
        done
    fi
}

function setup_custom_cni(){
    declare -A POD_CIDR=( ["cluster2"]="10.245.0.0/16" ["cluster3"]="10.246.0.0/16" )
    for i in 2 3; do
        if kubectl --context=cluster${i} wait --for=condition=Ready pods -l name=weave-net -n kube-system --timeout=60s > /dev/null 2>&1; then
            echo "Weave already deployed cluster${i}."
        else
            echo "Applying weave network in to cluster${i}..."
            kubectl --context=cluster${i} apply -f "https://cloud.weave.works/k8s/net?k8s-version=$(kubectl version | base64 | tr -d '\n')&env.IPALLOC_RANGE=${POD_CIDR[cluster${i}]}"
            echo "Waiting for weave-net pods to be ready cluster${i}..."
            kubectl --context=cluster${i} wait --for=condition=Ready pods -l name=weave-net -n kube-system --timeout=700s
            echo "Waiting for core-dns deployment to be ready cluster${i}..."
            kubectl --context=cluster${i} -n kube-system rollout status deploy/coredns --timeout=300s
        fi
    done
}

function setup_broker() {
    if kubectl --context=cluster1 get crd clusters.submariner.io > /dev/null 2>&1; then
        echo Submariner CRDs already exist, skipping broker creation...
    else
        echo Installing broker on cluster1.
        ../bin/subctl --kubeconfig ${PRJ_ROOT}/output/kind-config/dapper/kind-config-cluster1 deploy-broker --no-dataplane
    fi

    SUBMARINER_BROKER_URL=$(kubectl --context=cluster1 -n default get endpoints kubernetes -o jsonpath="{.subsets[0].addresses[0].ip}:{.subsets[0].ports[?(@.name=='https')].port}")
    SUBMARINER_BROKER_CA=$(kubectl --context=cluster1 -n ${SUBMARINER_BROKER_NS} get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='${SUBMARINER_BROKER_NS}-client')].data['ca\.crt']}")
    SUBMARINER_BROKER_TOKEN=$(kubectl --context=cluster1 -n ${SUBMARINER_BROKER_NS} get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='${SUBMARINER_BROKER_NS}-client')].data.token}"|base64 --decode)
}

function kind_import_images() {
    docker pull quay.io/submariner/submariner:latest
    docker tag quay.io/submariner/submariner:latest submariner:local
    docker pull quay.io/submariner/submariner-route-agent:latest
    docker tag quay.io/submariner/submariner-route-agent:latest submariner-route-agent:local
    docker tag quay.io/submariner/submariner-operator:dev submariner-operator:local

    for i in 2 3; do
        echo "Loading submariner images in to cluster${i}..."
        kind --name cluster${i} load docker-image submariner:local
        kind --name cluster${i} load docker-image submariner-route-agent:local
        kind --name cluster${i} load docker-image submariner-operator:local
    done
}

function create_subm_vars() {
  # FIXME A better name might be submariner-engine, but just kinda-matching submariner-<random hash> name used by Helm/upstream tests
  deployment_name=submariner
  operator_deployment_name=submariner-operator
  engine_deployment_name=submariner-engine
  routeagent_deployment_name=submariner-routeagent
  broker_deployment_name=submariner-k8s-broker

  clusterCIDR_cluster2=10.245.0.0/16
  clusterCIDR_cluster3=10.246.0.0/16
  serviceCIDR_cluster2=100.95.0.0/16
  serviceCIDR_cluster3=100.96.0.0/16
  natEnabled=false

  subm_engine_image_repo=local
  subm_engine_image_tag=local

  # FIXME: Actually act on this size request in controller
  subm_engine_size=3
  subm_colorcodes=blue
  subm_debug=false
  subm_broker=k8s
  ce_ipsec_debug=false
  ce_ipsec_ikeport=500
  ce_ipsec_nattport=4500

  subm_ns=submariner-operator
  subm_broker_ns=submariner-k8s-broker
}

function test_connection() {
    nginx_svc_ip_cluster3=$(kubectl --context=cluster3 get svc -l app=nginx-demo | awk 'FNR == 2 {print $3}')
    netshoot_pod=$(kubectl --context=cluster2 get pods -l app=netshoot | awk 'FNR == 2 {print $1}')

    echo "Testing connectivity between clusters - $netshoot_pod cluster2 --> $nginx_svc_ip_cluster3 nginx service cluster3"

    attempt_counter=0
    max_attempts=5
    until $(kubectl --context=cluster2 exec ${netshoot_pod} -- curl --output /dev/null -m 30 --silent --head --fail ${nginx_svc_ip_cluster3}); do
        if [[ ${attempt_counter} -eq ${max_attempts} ]];then
          echo "Max attempts reached, connection test failed!"
          exit 1
        fi
        attempt_counter=$(($attempt_counter+1))
    done
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
    set -o pipefail 

    cd ../test/e2e

    # Setup the KUBECONFIG env
    export KUBECONFIG=$(echo ${PRJ_ROOT}/output/kind-config/dapper/kind-config-cluster{1..3} | sed 's/ /:/g')

    go test -args -ginkgo.v -ginkgo.randomizeAllSpecs -ginkgo.reportPassed \
        -dp-context cluster2 -dp-context cluster3  \
        -report-dir ${DAPPER_SOURCE}/${DAPPER_OUTPUT}/junit 2>&1 | \
        tee ${DAPPER_SOURCE}/${DAPPER_OUTPUT}/e2e-tests.log
}

function cleanup {
    for i in 1 2 3; do
      if [[ $(kind get clusters | grep cluster${i} | wc -l) -gt 0  ]]; then
        kind delete cluster --name=cluster${i};
      fi
    done

    if [[ $(docker ps -qf status=exited | wc -l) -gt 0 ]]; then
        echo Cleaning containers...
        docker ps -qf status=exited | xargs docker rm -f
    fi
    if [[ $(docker images -qf dangling=true | wc -l) -gt 0 ]]; then
        echo Cleaning images...
        docker images -qf dangling=true | xargs docker rmi -f
    fi
#    if [[ $(docker images -q --filter=reference='submariner*:local' | wc -l) -gt 0 ]]; then
#        docker images -q --filter=reference='submariner*:local' | xargs docker rmi -f
#    fi
    if [[ $(docker volume ls -qf dangling=true | wc -l) -gt 0 ]]; then
        echo Cleaning volumes...
        docker volume ls -qf dangling=true | xargs docker volume rm -f
    fi
}

function dump_clusters_on_error {
    for i in 1 2 3; do
        echo "Dumping cluster information for cluster$i ==========================="
        kubectl --context=cluster$i cluster-info dump
        echo ""
    done
    exit 1
}

### Main ###

if [[ $1 = clean ]]; then
    cleanup
    exit 0
fi

if [[ $1 != keep ]]; then
    trap cleanup EXIT
fi

echo Starting with status: $1, k8s_version: $2.
PRJ_ROOT=$(git rev-parse --show-toplevel)
mkdir -p ${PRJ_ROOT}/output/kind-config/dapper/ ${PRJ_ROOT}/output/kind-config/local-dev/
SUBMARINER_BROKER_NS=submariner-k8s-broker
# FIXME: This can change and break re-running deployments
SUBMARINER_PSK=$(cat /dev/urandom | LC_CTYPE=C tr -dc 'a-zA-Z0-9' | fold -w 64 | head -n 1)
export KUBECONFIG=$(echo ${PRJ_ROOT}/output/kind-config/dapper/kind-config-cluster{1..3} | sed 's/ /:/g')

kind_clusters "$@"
setup_custom_cni

kind_import_images
setup_broker

context=cluster1
kubectl config use-context $context

# Import functions for testing with Operator
# NB: These are also used to verify non-Operator deployments, thereby asserting the two are mostly equivalent
. kind-e2e/lib_operator_verify_subm.sh

create_subm_vars
verify_subm_broker_secrets

. kind-e2e/lib_operator_deploy_subm.sh

trap dump_clusters_on_error ERR

for i in 2 3; do
    context=cluster$i
    kubectl config use-context $context

    # Create CRDs required as prerequisite submariner-engine
    # TODO: Eventually OLM and subctl should handle this
    create_subm_endpoints_crd
    verify_endpoints_crd
    create_subm_clusters_crd
    verify_clusters_crd

    # Add SubM gateway labels
    add_subm_gateway_label
    # Verify SubM gateway labels
    verify_subm_gateway_label

    # Deploy SubM Operator
    ../bin/subctl join --image submariner-operator:local \
                        --kubeconfig ${PRJ_ROOT}/output/kind-config/dapper/kind-config-$context \
                        broker-info.subm
    # Verify SubM CRD
    verify_subm_crd
    # Verify SubM Operator
    verify_subm_operator
    # Verify SubM Operator pod
    verify_subm_op_pod
    # Verify SubM Operator container
    verify_subm_operator_container

    # FIXME: Rename all of these submariner-engine or engine, vs submariner
    # Create SubM CR
    create_subm_cr
    # Deploy SubM CR
    deploy_subm_cr
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
    # Verify Routeagent secrets
    verify_subm_routeagent_secrets
done

deploy_netshoot_cluster2
deploy_nginx_cluster3

test_connection

#TODO(mangelajo): test_with_e2e_tests disabled for now, the day we have an e2e testing image
#                 we can use it, or we can run our own e2e tests (operator specifics) when
#                we have them.
#test_with_e2e_tests

if [[ $1 = keep ]]; then
    echo "your 3 virtual clusters are deployed and working properly with your local"
    echo "submariner source code, and can be accessed with:"
    echo ""
    echo "export KUBECONFIG=\$(echo \$(git rev-parse --show-toplevel)/output/kind-config/local-dev/kind-config-cluster{1..3} | sed 's/ /:/g')"
    echo ""
    echo "$ kubectl config use-context cluster1 # or cluster2, cluster3.."
    echo ""
    echo "to cleanup, just run: make e2e status=clean"
fi
