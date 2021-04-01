#!/bin/bash

# This should only be sourced
if [ "${0##*/}" = "lib_subctl_gather_test.sh" ]; then
    echo "Don't run me, source me" >&2
    exit 1
fi

function test_subctl_gather() {
  out_dir=/tmp/subctl-gather-output
  rm -rf $out_dir
  mkdir $out_dir

  ${DAPPER_SOURCE}/bin/subctl gather --kubeconfig ${KUBECONFIGS_DIR}/kind-config-${cluster} --dir $out_dir

  # connectivity
  validate_resource_files $subm_ns 'endpoints.submariner.io' 'Endpoint'
  validate_resource_files $subm_ns 'clusters.submariner.io' 'Cluster'
  validate_resource_files $subm_ns 'gateways.submariner.io' 'Gateway'

  validate_pod_log_files $subm_ns $gateway_deployment_name
  validate_pod_log_files $subm_ns $routeagent_deployment_name
  validate_pod_log_files $subm_ns $globalnet_deployment_name
  validate_pod_log_files $subm_ns 'submariner-networkplugin-syncer'

  # operator
  validate_resource_files $subm_ns 'submariners' 'Submariner'
  validate_resource_files $subm_ns 'servicediscoveries' 'ServiceDiscovery'
  validate_resource_files $subm_ns 'daemonsets' 'DaemonSet' '-l app=submariner-gateway'
  validate_resource_files $subm_ns 'daemonsets' 'DaemonSet' '-l app=submariner-routeagent'
  validate_resource_files $subm_ns 'daemonsets' 'DaemonSet' '-l app=submariner-globalnet'
  validate_resource_files $subm_ns 'deployments' 'Deployment' '-l app=submariner-networkplugin-syncer'
  validate_resource_files $subm_ns 'deployments' 'Deployment' '-l app=submariner-lighthouse-agent'
  validate_resource_files $subm_ns 'deployments' 'Deployment' '-l app=submariner-lighthouse-coredns'
}

function validate_pod_log_files() {
  local ns=$1
  local label=$2

  pod_names=$(kubectl get pods --namespace=$ns -l app=$label -o=jsonpath='{.items..metadata.name}')
  read -ra pod_names_array <<< "$pod_names"

  for pod_name in "${pod_names_array[@]}"; do
    file=$out_dir/${cluster}_$pod_name.log
    cat $file
  done
}

function validate_resource_files() {
  local ns=$1
  local resource=$2
  local kind=$3
  local selector=$4

  names=$(kubectl get $resource --namespace=$ns $selector -o=jsonpath='{.items..metadata.name}')
  read -ra names_array <<< "$names"

  short_res=$(echo $resource | awk -F. '{ print $1 }')

  for name in "${names_array[@]}"; do
    file=$out_dir/${cluster}_${short_res}_${ns}_${name}.yaml
    cat $file

    kind_count=$(grep "kind: $kind$" $file | wc -l)
    if [[ $kind_count != "1" ]]; then
      echo "Expected 1 kind: $kind"
     return 1
    fi

    res_name=$(grep "name: $name$" $file)
    if [[ $res_name == "" ]]; then
      echo "Expected resource name: $name"
     return 1
    fi
  done
}

