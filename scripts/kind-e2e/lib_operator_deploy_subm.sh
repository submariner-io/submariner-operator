#!/bin/bash
# This should only be sourced
if [ "${0##*/}" = "lib_operator_deploy_subm.sh" ]; then
    echo "Don't run me, source me" >&2
    exit 1
fi

openapi_checks_enabled=false

function add_subm_gateway_label() {
  kubectl label node $context-worker "submariner.io/gateway=true" --overwrite
}

function deploy_netshoot_cluster2() {
    kubectl config use-context cluster2
    echo Deploying netshoot on cluster2 worker: ${worker_ip}
    kubectl apply -f ${DAPPER_SOURCE}/scripts/kind-e2e/netshoot.yaml
    echo Waiting for netshoot pods to be Ready on cluster2.
    kubectl rollout status deploy/netshoot --timeout=120s

    # TODO: Add verifications
}

function deploy_nginx_cluster3() {
    kubectl config use-context cluster3
    echo Deploying nginx on cluster3 worker: ${worker_ip}
    kubectl apply -f ${DAPPER_SOURCE}/scripts/kind-e2e/nginx-demo.yaml
    echo Waiting for nginx-demo deployment to be Ready on cluster3.
    kubectl rollout status deploy/nginx-demo --timeout=120s

    # TODO: Add verifications
    # TODO: Do this with nginx operator?
}
