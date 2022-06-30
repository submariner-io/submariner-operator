#!/usr/bin/env bash

## Process command line flags ##

source ${SCRIPTS_DIR}/lib/shflags
DEFINE_string 'settings' '' "Settings YAML file to customize cluster deployments"
DEFINE_boolean 'globalnet' false "Deploy with operlapping CIDRs (set to 'true' to enable)"
FLAGS "$@" || exit $?
eval set -- "${FLAGS_ARGV}"

settings="${FLAGS_settings}"
[[ "${FLAGS_globalnet}" = "${FLAGS_TRUE}" ]] && globalnet=true || globalnet=false

set -em -o pipefail

source ${SCRIPTS_DIR}/lib/debug_functions
source ${SCRIPTS_DIR}/lib/utils
source ${SCRIPTS_DIR}/lib/deploy_funcs
source ${SCRIPTS_DIR}/lib/deploy_operator

### Variables ###

[[ ! ${DEPLOY_ARGS} =~ "service-discovery" ]] || lighthouse=true
timeout="2m" # Used by deploy_resource

### Main ###

load_settings
declare_kubeconfig

# Import functions for testing with Operator
# NB: These are also used to verify non-Operator deployments, thereby asserting the two are mostly equivalent
. ${DAPPER_SOURCE}/scripts/test/lib_operator_verify_subm.sh

create_subm_vars
with_context "$broker" broker_vars

with_context "$broker" verify_subm_broker_secrets

run_subm_clusters verify_subm_deployed

if [[ "$lighthouse" == "true" ]]; then
    verify="--only service-discovery"
else
    verify="--only connectivity"
fi

# run dataplane E2E tests between the two clusters
subctl verify ${verify} --submariner-namespace=$subm_ns \
    --verbose --connection-timeout 20 --connection-attempts 4 \
    --kubecontexts cluster1,cluster2

${SCRIPTS_DIR}/e2e.sh "$@"

print_clusters_message

