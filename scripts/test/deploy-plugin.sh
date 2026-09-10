#!/usr/bin/env bash

set -eo pipefail

function post_deploy() {
    echo "Installing cert-manager on all clusters..."
    run_all_clusters install_cert_manager
}

function install_cert_manager() {
    local cert_manager_version="${CERT_MANAGER_VERSION:-v1.17.4}"

    # shellcheck disable=SC2154 # cluster is set by run_all_clusters
    echo "Installing cert-manager ${cert_manager_version} on cluster ${cluster}..."

    kubectl apply -f "https://github.com/cert-manager/cert-manager/releases/download/${cert_manager_version}/cert-manager.yaml"

    kubectl wait --for=condition=Available --timeout=300s deployment/cert-manager -n cert-manager
    kubectl wait --for=condition=Available --timeout=300s deployment/cert-manager-webhook -n cert-manager
    kubectl wait --for=condition=Available --timeout=300s deployment/cert-manager-cainjector -n cert-manager

    echo "cert-manager is ready on cluster ${cluster}"
}
