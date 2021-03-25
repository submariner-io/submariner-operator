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

  validate_pod_log_files $subm_ns '-l app=submariner-gateway'
  validate_pod_log_files $subm_ns '-l app=submariner-routeagent'
  validate_pod_log_files $subm_ns '-l app=submariner-globalnet'
  validate_pod_log_files $subm_ns '-l app=submariner-networkplugin-syncer'

  # operator
  validate_resource_files $subm_ns 'submariners' 'Submariner'
  validate_resource_files $subm_ns 'servicediscoveries' 'ServiceDiscovery'
  validate_resource_files $subm_ns 'daemonsets' 'DaemonSet' '-l app=submariner-gateway'
  validate_resource_files $subm_ns 'daemonsets' 'DaemonSet' '-l app=submariner-routeagent'
  validate_resource_files $subm_ns 'daemonsets' 'DaemonSet' '-l app=submariner-globalnet'
  validate_resource_files $subm_ns 'deployments' 'Deployment' '-l app=submariner-networkplugin-syncer'
  validate_resource_files $subm_ns 'deployments' 'Deployment' '-l app=submariner-lighthouse-agent'
  validate_resource_files $subm_ns 'deployments' 'Deployment' '-l app=submariner-lighthouse-coredns'

  # Service Discovery
  validate_resource_files all 'serviceexports.multicluster.x-k8s.io' 'ServiceExport'
  validate_resource_files all 'serviceimports.multicluster.x-k8s.io' 'ServiceImport'
  validate_resource_files all 'endpointslices.discovery.k8s.io' 'EndpointSlice'
  validate_resource_files $subm_ns 'configmaps' 'ConfigMap' '-l component=submariner-lighthouse'
  validate_resource_files kube-system 'configmaps' 'ConfigMap' '--field-selector metadata.name=coredns'

  validate_pod_log_files $subm_ns '-l component=submariner-lighthouse'
  validate_pod_log_files kube-system '-l k8s-app=kube-dns'

  # Broker
  with_context "$broker" validate_broker_resources
}

function validate_pod_log_files() {
  local ns=$1
  local selector=$2
  local nsarg="--namespace=${ns}"

  if [[ "$ns" == "all" ]]; then
    nsarg="-A"
  fi
  pod_names=$(kubectl get pods $nsarg $selector -o=jsonpath='{.items..metadata.name}')
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
  local cluster_name=$5

  if [[ $cluster_name == "" ]]; then
    cluster_name=${cluster}
  fi

  names=$(kubectl get $resource --namespace=$ns $selector -o=jsonpath='{.items..metadata.name}')
  read -ra names_array <<< "$names"

  short_res=$(echo $resource | awk -F. '{ print $1 }')

  for name in "${names_array[@]}"; do
    file=$out_dir/${cluster_name}_${short_res}_${ns}_${name}.yaml
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

function validate_broker_resources() {
  validate_resource_files $SUBMARINER_BROKER_NS 'endpoints.submariner.io' 'Endpoint' '' 'broker'
  validate_resource_files $SUBMARINER_BROKER_NS 'clusters.submariner.io' 'Cluster' '' 'broker'
  validate_resource_files $SUBMARINER_BROKER_NS 'serviceimports.multicluster.x-k8s.io' 'ServiceImport' '' 'broker'
  validate_resource_files $SUBMARINER_BROKER_NS 'endpointslices.discovery.k8s.io' 'EndpointSlice' '' 'broker'
}
