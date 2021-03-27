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

  validate_resources_file $subm_ns 'endpoints.submariner.io' 'Endpoint' "$out_dir/${cluster}-endpoints.yaml"
  validate_resources_file $subm_ns 'clusters.submariner.io' 'Cluster' "$out_dir/${cluster}-clusters.yaml"
  validate_resources_file $subm_ns 'gateways.submariner.io' 'Gateway' "$out_dir/${cluster}-gateways.yaml"

  validate_pod_log_files $subm_ns $gateway_deployment_name
  validate_pod_log_files $subm_ns $routeagent_deployment_name
}

function validate_pod_log_files() {
  local ns=$1
  local label=$2

  pod_names=$(kubectl get pods --namespace=$ns -l app=$label -o=jsonpath='{.items..metadata.name}')
  read -ra pod_names_array <<< "$pod_names"

  for pod_name in "${pod_names_array[@]}"; do
    file=$out_dir/${cluster}-$pod_name.log
    cat $file
  done
}

function validate_resources_file() {
  local ns=$1
  local resource=$2
  local kind=$3
  local file=$4

  exp_num=$(kubectl get $resource --namespace=$ns -o=yaml | grep "kind: $kind$" | wc -l)

  cat $file
  actual_num=$(grep "kind: $kind$" $file | wc -l)
  if [[ $exp_num != $actual_num ]]; then
     echo "Expected $exp_num $resource but got $actual_num"
     return 1
  fi
}

