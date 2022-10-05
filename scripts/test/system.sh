#!/bin/bash

. "${SCRIPTS_DIR}/lib/debug_functions"
. "${SCRIPTS_DIR}/lib/utils"

set -eo pipefail

### Functions ###

function verify_subm_gateway_label() {
  kubectl get node "${cluster}-worker" -o jsonpath='{.metadata.labels}' | grep submariner.io/gateway:true
}

function broker_vars() {
    SUBMARINER_BROKER_URL=$(kubectl -n default get endpoints kubernetes -o jsonpath="{.subsets[0].addresses[0].ip}:{.subsets[0].ports[?(@.name=='https')].port}")
    SUBMARINER_BROKER_CA=$(kubectl -n "${SUBMARINER_BROKER_NS}" get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='${SUBMARINER_BROKER_NS}-client')].data['ca\.crt']}")
    #SUBMARINER_BROKER_TOKEN=$(kubectl -n "${SUBMARINER_BROKER_NS}" get secrets -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='${SUBMARINER_BROKER_NS}-client')].data.token}"|base64 --decode)
}

function create_subm_vars() {
    deployment_name=submariner
    operator_deployment_name=submariner-operator
    gateway_deployment_name=submariner-gateway
    routeagent_deployment_name=submariner-routeagent
    broker_deployment_name=submariner-k8s-broker

    declare_cidrs
    natEnabled=false

    # The version is set in api/v1alpha1/versions.go but `subctl` overrides it to the base branch
    subm_gateway_image_tag=${BASE_BRANCH}
    subm_gateway_image_repo=$(git grep DefaultRepo api/v1alpha1/versions.go | cut -f2 -d'"')

    subm_debug=false
    subm_broker=k8s
    ce_ipsec_debug=false
    ce_ipsec_nattport=4500

    subm_ns=submariner-operator
    SUBMARINER_BROKER_NS=submariner-k8s-broker
}

function verify_subm_operator() {
  # Verify SubM namespace (ignore SubM Broker ns)
  kubectl get ns "$subm_ns"

  # Verify SubM Operator CRD
  kubectl get crds submariners.submariner.io
  kubectl api-resources | grep submariners

  # Verify SubM Operator SA
  kubectl get sa --namespace="$subm_ns" submariner-operator

  # Verify SubM Operator role
  kubectl get roles --namespace="$subm_ns" submariner-operator || \
  kubectl get roles -n "$subm_ns" -l olm.owner.namespace=submariner-operator

  # Verify SubM Operator role binding
  kubectl get rolebindings --namespace="$subm_ns" submariner-operator || \
  kubectl get rolebindings -n "$subm_ns" -l olm.owner.namespace=submariner-operator

  # Verify SubM Operator deployment
  kubectl get deployments --namespace="$subm_ns" submariner-operator
}

function verify_subm_deployed() {

    # Verify shared CRDs
    verify_endpoints_crd
    verify_clusters_crd

    # Verify SubM CRD
    verify_subm_crd
    # Verify SubM Operator
    verify_subm_operator
    # Verify SubM Operator pod
    verify_subm_op_pod
    # We don't verify the operator container, it only contains the
    # operator binary

    # Verify SubM CR
    verify_subm_cr
    # Verify Subm CR status
    verify_subm_cr_status_with_retries
    # Verify SubM Gateway Deployment
    verify_daemonset submariner-gateway
    # Verify SubM Gateway Pod
    verify_subm_gateway_pod
    # Verify SubM Gateway container
    verify_subm_gateway_container
    # Verify Gateway secrets
    verify_subm_gateway_secrets

    # Verify SubM Routeagent DaemonSet
    verify_daemonset submariner-routeagent
    # Verify SubM Routeagent Pods
    verify_subm_routeagent_pod
    # Verify SubM Routeagent container
    verify_subm_routeagent_container

    if [[ "$GLOBALNET" == true ]]; then
        verify_daemonset submariner-globalnet
    fi

    verify_network_plugin_syncer
}

# Uses `jq` to extract the content using the filter given, and matches it to the expected value
# Make sure $json_file is set to point to the file to check
function validate_equals() {
  local json_filter=$1
  local expected=$2
  local actual
  actual=$(jq -r -M "$json_filter" "$json_file")
  if [[ "$expected" != "$actual" ]]; then
     echo "Expected ${expected@Q} but got ${actual@Q}"
     return 1
  fi
}

function validate_not_equals() {
  local json_filter=$1
  local expected=$2
  [[ "$expected" != $(jq -r -M "$json_filter" "$json_file") ]]
}


function validate_crd() {
  local crd_name=$1
  local spec_name=$2

  # Verify presence of CRD
  kubectl get crds "$crd_name"

  # Show full CRD
  json_file="/tmp/${crd_name}.${cluster}.json"
  kubectl get crd "$crd_name" -o json | tee "$json_file"

  # Verify details of CRD
  validate_equals '.metadata.name' "$crd_name"
  validate_equals '.spec.scope' 'Namespaced'
  validate_equals '.spec.group' 'submariner.io'
  validate_equals '.spec.names.kind' "$spec_name"
}

function verify_subm_crd() {
  validate_crd 'submariners.submariner.io' 'Submariner'
}

function verify_endpoints_crd() {
  validate_crd 'endpoints.submariner.io' 'Endpoint'
  validate_equals '.status.acceptedNames.kind' 'Endpoint'
}

function verify_clusters_crd() {
  validate_crd 'clusters.submariner.io' 'Cluster'
  validate_equals '.status.acceptedNames.kind' 'Cluster'
}

# retries are necessary for the status field, which can take some seconds to fill up
# properly by the operator
function verify_subm_cr_status_with_retries() {
  function verify_subm_cr_status_() {
    if ! verify_subm_cr_status; then
      sleep 5 && return 1
    fi
    return 0
  }
  with_retries 5 verify_subm_cr_status_
}

function verify_subm_cr_status() {

  json_file=/tmp/${deployment_name}.${cluster}.json
  kubectl get submariner "$deployment_name" --namespace=$subm_ns -o json > "$json_file"

  validate_equals '.status.serviceCIDR' "${service_CIDRs[$cluster]}"
  validate_equals '.status.clusterCIDR' "${cluster_CIDRs[$cluster]}"

}

function verify_subm_cr() {
  # TODO: Use $gateway_deployment_name here?

  # Verify SubM CR presence
  kubectl get submariner --namespace=$subm_ns | grep "$deployment_name"

  # Show full SubM CR
  json_file=/tmp/${deployment_name}.${cluster}.json
  kubectl get submariner "$deployment_name" --namespace=$subm_ns -o json | tee "$json_file"

  validate_equals '.metadata.namespace' "$subm_ns"
  validate_equals '.apiVersion' 'submariner.io/v1alpha1'
  validate_equals '.kind' 'Submariner'
  validate_equals '.metadata.name' "$deployment_name"
  validate_equals '.spec.brokerK8sApiServer' "$SUBMARINER_BROKER_URL"
  # TODO: every cluster must have it's own token / SA (not working when using bundle/acm)
  # validate_not_equals '.spec.brokerK8sApiServerToken' $SUBMARINER_BROKER_TOKEN
  validate_equals '.spec.brokerK8sCA' "$SUBMARINER_BROKER_CA"
  validate_equals '.spec.brokerK8sRemoteNamespace' "$SUBMARINER_BROKER_NS"
  validate_equals '.spec.ceIPSecDebug' "$ce_ipsec_debug"
  validate_equals '.spec.ceIPSecNATTPort' "$ce_ipsec_nattport"
  validate_equals '.spec.repository' "$subm_gateway_image_repo"
  validate_equals '.spec.version' "$subm_gateway_image_tag"
  validate_equals '.spec.broker' "$subm_broker"
  echo "Generated cluster id: $(jq -r '.spec.clusterID' "$json_file")"
  validate_equals '.spec.debug' "$subm_debug"
  validate_equals '.spec.namespace' "$subm_ns"
  validate_equals '.spec.natEnabled' "$natEnabled"

}

function verify_subm_op_pod() {
  subm_operator_pod_name=$(kubectl get pods --namespace="$subm_ns" -l name="$operator_deployment_name" -o=jsonpath='{.items..metadata.name}')
  if [[ -z "${subm_operator_pod_name}" ]]; then
    subm_operator_pod_name=$(kubectl get pods --namespace="$subm_ns" -l control-plane="$operator_deployment_name" -o=jsonpath='{.items..metadata.name}')
  fi

  # Show SubM Operator pod info
  kubectl get pod "$subm_operator_pod_name" --namespace="$subm_ns" -o json

  # Verify SubM Operator pod status
  kubectl get pod "$subm_operator_pod_name" --namespace="$subm_ns" -o jsonpath='{.status.phase}' | grep Running

  # Show SubM Operator pod logs
  kubectl logs "$subm_operator_pod_name" --namespace="$subm_ns"

  # TODO: Verify logs?

  json_file=/tmp/${subm_operator_pod_name}.${cluster}.json
  kubectl get pod "$subm_operator_pod_name" --namespace="$subm_ns" -o json | tee "$json_file"

  validate_pod_container_equals 'image' "localhost:5000/submariner-operator:local"
  validate_pod_container_has 'command' 'submariner-operator'
}

function validate_pod_container_equals() {
  validate_equals ".spec.containers[].${1}" "$2"
}

function validate_pod_container_has() {
  local json_filter=.spec.containers[].${1}
  local expected=$2
  [[ $(jq -r -M "$json_filter" "$json_file") =~ $expected ]]
}

function validate_pod_container_env() {
  local var_name=$1
  local expected=$2
  [[ $(jq -r -M ".spec.containers[].env[] | select(.name==\"${var_name}\").value" "$json_file") = "$expected" ]]
}

function verify_subm_gateway_pod() {
  kubectl wait --for=condition=Ready pods -l app=$gateway_deployment_name --timeout=120s --namespace=$subm_ns

  subm_gateway_pod_name=$(kubectl get pods --namespace=$subm_ns -l app=$gateway_deployment_name -o=jsonpath='{.items..metadata.name}')

  json_file=/tmp/${subm_gateway_pod_name}.${cluster}.json
  kubectl get pod "$subm_gateway_pod_name" --namespace="$subm_ns" -o json | tee "$json_file"

  validate_pod_container_equals 'image' "${subm_gateway_image_repo}/submariner-gateway:${subm_gateway_image_tag}"
  validate_pod_container_has 'securityContext.capabilities.add' 'net_admin'
  validate_pod_container_equals 'securityContext.allowPrivilegeEscalation' 'true'
  validate_pod_container_equals 'securityContext.privileged' 'true'
  validate_pod_container_equals 'securityContext.readOnlyRootFilesystem' 'false'
  validate_pod_container_equals 'securityContext.runAsNonRoot' 'false'
  validate_pod_container_has 'command' 'submariner.sh'

  jq -r '.spec.containers[].env' "$json_file"
  validate_pod_container_env 'SUBMARINER_NAMESPACE' "$subm_ns"
  validate_pod_container_env 'SUBMARINER_SERVICECIDR' "${service_CIDRs[$cluster]}"
  validate_pod_container_env 'SUBMARINER_CLUSTERCIDR' "${cluster_CIDRs[$cluster]}"
  validate_pod_container_env 'SUBMARINER_DEBUG' "$subm_debug"
  validate_pod_container_env 'SUBMARINER_NATENABLED' "$natEnabled"
  validate_pod_container_env 'SUBMARINER_BROKER' "$subm_broker"
  validate_pod_container_env 'BROKER_K8S_APISERVER' "$SUBMARINER_BROKER_URL"
  validate_pod_container_env 'BROKER_K8S_REMOTENAMESPACE' "$SUBMARINER_BROKER_NS"
  validate_pod_container_env 'BROKER_K8S_CA' "$SUBMARINER_BROKER_CA"
  validate_pod_container_env 'CE_IPSEC_DEBUG' "$ce_ipsec_debug"
  validate_pod_container_env 'CE_IPSEC_NATTPORT' "$ce_ipsec_nattport"

  validate_equals '.spec.serviceAccount' 'submariner-gateway'
  validate_equals '.status.phase' 'Running'
  validate_equals '.metadata.namespace' "$subm_ns"
}

function daemonset_created() {
  kubectl get DaemonSet "$daemonset_name" -n "$subm_ns" > /dev/null 2>&1
}

function daemonset_deployed() {
  local desiredNumberScheduled numberReady
  desiredNumberScheduled=$(kubectl get DaemonSet "$daemonset_name" -n "$subm_ns" -o jsonpath='{.status.desiredNumberScheduled}')
  numberReady=$(kubectl get DaemonSet "$daemonset_name" -n "$subm_ns" -o jsonpath='{.status.numberReady}')
  [[ "$numberReady" = "$desiredNumberScheduled" ]]
}

function verify_daemonset() {
  local daemonset_name=$1

  # Simple verification to ensure that the daemonset has been created and becomes ready
  with_retries 60 daemonset_created
  with_retries 120 daemonset_deployed
}

validate_pod_container_volume_mount() {
  local volume_path=$1
  local volume_name=$2
  local read_only=$3
  [[ $(jq -r ".spec.containers[].volumeMounts[] | select(.name==\"${volume_name}\").mountPath" "$json_file") = "$volume_path" ]]
  [[ $(jq -r ".spec.containers[].volumeMounts[] | select(.name==\"${volume_name}\").readOnly" "$json_file") = "$read_only" ]]
}

function verify_subm_routeagent_pod() {
  kubectl wait --for=condition=Ready pods -l app=$routeagent_deployment_name --timeout=120s --namespace=$subm_ns

  # Loop tests over all routeagent pods
  subm_routeagent_pod_names=$(kubectl get pods --namespace=$subm_ns -l app=$routeagent_deployment_name -o=jsonpath='{.items..metadata.name}')
  # Globing-safe method, but -a flag gives me trouble in ZSH for some reason
  read -ra subm_routeagent_pod_names_array <<< "$subm_routeagent_pod_names"
  # TODO: Fail if there are zero routeagent pods
  for subm_routeagent_pod_name in "${subm_routeagent_pod_names_array[@]}"; do
    echo "Testing Submariner routeagent pod $subm_routeagent_pod_name"
    json_file=/tmp/${subm_gateway_pod_name}.${cluster}.json
    kubectl get pod "$subm_routeagent_pod_name" --namespace="$subm_ns" -o json | tee "$json_file"
    validate_pod_container_equals 'image' "${subm_gateway_image_repo}/submariner-route-agent:$subm_gateway_image_tag"
    validate_pod_container_has 'securityContext.capabilities.add' 'ALL'
    validate_pod_container_equals 'securityContext.allowPrivilegeEscalation' 'true'
    validate_pod_container_equals 'securityContext.privileged' 'true'
    validate_pod_container_equals 'securityContext.readOnlyRootFilesystem' 'false'
    validate_pod_container_equals 'securityContext.runAsNonRoot' 'false'
    validate_pod_container_has 'command' 'submariner-route-agent.sh'

    jq -r '.spec.containers[].env' "$json_file"
    validate_pod_container_env 'SUBMARINER_NAMESPACE' "$subm_ns"
    validate_pod_container_env 'SUBMARINER_DEBUG' "$subm_debug"
    validate_pod_container_env 'SUBMARINER_SERVICECIDR' "${service_CIDRs[$cluster]}"
    validate_pod_container_env 'SUBMARINER_CLUSTERCIDR' "${cluster_CIDRs[$cluster]}"

    validate_equals '.spec.serviceAccount' 'submariner-routeagent'

    validate_equals '.status.phase' 'Running'
    validate_equals '.metadata.namespace' "$subm_ns"
    validate_equals '.spec.terminationGracePeriodSeconds' '1'
  done
}

function verify_subm_gateway_container() {
  subm_gateway_pod_name=$(kubectl get pods --namespace=$subm_ns -l app=$gateway_deployment_name -o=jsonpath='{.items..metadata.name}')

  # Show SubM Gateway pod environment variables
  env_file=/tmp/${subm_gateway_pod_name}.env
  kubectl exec "$subm_gateway_pod_name" --namespace="$subm_ns" -- env | tee "$env_file"

  # Verify SubM Gateway pod environment variables
  grep "BROKER_K8S_APISERVER=$SUBMARINER_BROKER_URL" "$env_file"
  grep "SUBMARINER_NAMESPACE=$subm_ns" "$env_file"
  grep "SUBMARINER_BROKER=$subm_broker" "$env_file"
  grep "BROKER_K8S_CA=$SUBMARINER_BROKER_CA" "$env_file"
  grep "CE_IPSEC_DEBUG=$ce_ipsec_debug" "$env_file"
  grep "SUBMARINER_DEBUG=$subm_debug" "$env_file"
  grep "BROKER_K8S_REMOTENAMESPACE=$SUBMARINER_BROKER_NS" "$env_file"
  grep "SUBMARINER_SERVICECIDR=${service_CIDRs[$cluster]}" "$env_file"
  grep "SUBMARINER_CLUSTERCIDR=${cluster_CIDRs[$cluster]}" "$env_file"
  grep "SUBMARINER_NATENABLED=$natEnabled" "$env_file"
  grep "HOME=/root" "$env_file"

  if kubectl exec "$subm_gateway_pod_name" --namespace="$subm_ns" -- command -v command; then
    # Verify the gateway binary is in the expected place and in PATH
    kubectl exec "$subm_gateway_pod_name" --namespace="$subm_ns" -- command -v submariner-gateway | grep /usr/local/bin/submariner-gateway

    # Verify the gateway entry script is in the expected place and in PATH
    kubectl exec "$subm_gateway_pod_name" --namespace="$subm_ns" -- command -v submariner.sh | grep /usr/local/bin/submariner.sh
  elif kubectl exec "$subm_gateway_pod_name" --namespace="$subm_ns" -- which which; then
    # Verify the gateway binary is in the expected place and in PATH
    kubectl exec "$subm_gateway_pod_name" --namespace="$subm_ns" -- which submariner-gateway | grep /usr/local/bin/submariner-gateway

    # Verify the gateway entry script is in the expected place and in PATH
    kubectl exec "$subm_gateway_pod_name" --namespace="$subm_ns" -- which submariner.sh | grep /usr/local/bin/submariner.sh
  fi
}

function verify_subm_routeagent_container() {
  # Loop tests over all routeagent pods
  subm_routeagent_pod_names=$(kubectl get pods --namespace=$subm_ns -l app=$routeagent_deployment_name -o=jsonpath='{.items..metadata.name}')
  # Globing-safe method, but -a flag gives me trouble in ZSH for some reason
  read -ra subm_routeagent_pod_names_array <<<"$subm_routeagent_pod_names"
  # TODO: Fail if there are zero routeagent pods
  for subm_routeagent_pod_name in "${subm_routeagent_pod_names_array[@]}"; do
    echo "Testing Submariner routeagent container $subm_routeagent_pod_name"

    # Show SubM Routeagent pod environment variables
    env_file="/tmp/${subm_routeagent_pod_name}.env"
    kubectl exec "$subm_routeagent_pod_name" --namespace="$subm_ns" -- env | tee "$env_file"

    # Verify SubM Routeagent pod environment variables
    grep "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin" "$env_file"
    grep "SUBMARINER_NAMESPACE=$subm_ns" "$env_file"
    grep "SUBMARINER_DEBUG=$subm_debug" "$env_file"
    grep "SUBMARINER_SERVICECIDR=${service_CIDRs[$cluster]}" "$env_file"
    grep "SUBMARINER_CLUSTERCIDR=${cluster_CIDRs[$cluster]}" "$env_file"
    grep "HOME=/root" "$env_file"

    # Verify the routeagent binary is in the expected place and in PATH
    kubectl exec "$subm_routeagent_pod_name" --namespace="$subm_ns" -- command -v submariner-route-agent | grep /usr/local/bin/submariner-route-agent

    # Verify the routeagent entry script is in the expected place and in PATH
    kubectl exec "$subm_routeagent_pod_name" --namespace="$subm_ns" -- command -v submariner-route-agent.sh | grep /usr/local/bin/submariner-route-agent.sh
  done
}

function wait_for_secrets() {
  local secret_ns=$1
  for i in {1..30}; do
    if kubectl get secret -n "$secret_ns" | grep -q submariner-; then
      return
    fi
  done

  echo "Timeout waiting for SubM Secret creation"
  exit 1
}

function verify_secrets() {
  local secret_ns=$1
  local deployment_name=$2
  local expected_ca_crt=$3
  wait_for_secrets "$secret_ns"

  # Show all the secrets
  kubectl get secrets -n "$secret_ns"

  local secret_names
  secret_names=$(kubectl get secrets -n "$secret_ns" -o jsonpath="{.items[?(@.metadata.annotations['kubernetes\.io/service-account\.name']=='$deployment_name')].metadata.name}")
  if [[ -z "$secret_names" ]]; then
    echo "Failed to find the secret's name"
    exit 1
  fi

  local one_ca_crt_ok=false
  # Show all details of the secrets
  for secret_name in $secret_names; do
    json_file="/tmp/$secret_name.${cluster}.json"
    kubectl get secret "$secret_name" -n "$secret_ns" -o json | tee "$json_file"

    # Verify details of the secrets
    validate_equals '.kind' 'Secret'
    validate_equals '.type' "kubernetes.io/service-account-token"
    validate_equals '.metadata.name' "$secret_name"
    validate_equals '.metadata.namespace' "$secret_ns"
    if [[ $(jq -r -M '.data["ca.crt"]' "$json_file") =~ $expected_ca_crt ]]; then
      one_ca_crt_ok=true
    fi
  done
  [[ "${one_ca_crt_ok}" = "true" ]]
}

function verify_subm_broker_secrets() {
  verify_secrets "$SUBMARINER_BROKER_NS" "$broker_deployment_name-client" "$SUBMARINER_BROKER_CA"
}

function verify_subm_gateway_secrets() {
  # FIXME: There seems to be a strange error where the CA substantially match, but eventually actually are different
  verify_secrets "$subm_ns" "$operator_deployment_name" "${SUBMARINER_BROKER_CA:0:50}"
}

function verify_network_plugin_syncer {
   # Verify service account
  kubectl get sa --namespace=$subm_ns submariner-networkplugin-syncer

  # Verify cluster reole
  kubectl get clusterrole submariner-networkplugin-syncer

  # Verify cluster role binding
  kubectl get clusterrolebinding submariner-networkplugin-syncer
}

function deploy_env_once() {
    if with_context "${clusters[0]}" kubectl wait --for condition=Ready pods -l app=submariner-gateway -n "${subm_ns}" --timeout=3s > /dev/null 2>&1; then
        echo "Submariner already deployed, skipping deployment..."
        return
    fi

    make deploy SETTINGS="$SETTINGS" using="${USING}"
    declare_kubeconfig
}

### Main ###

SETTINGS="${DAPPER_SOURCE}/.shipyard.system.yml"
load_settings
create_subm_vars

declare_kubeconfig
deploy_env_once

with_context "$broker" broker_vars

with_context "$broker" verify_subm_broker_secrets

run_subm_clusters verify_subm_deployed

