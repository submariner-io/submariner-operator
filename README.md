# Submariner Operator

The submariner operator installs the submariner components on a Kubernetes cluster.

It's available on [OperatorHub:submariner](https://operatorhub.io/operator/submariner).

## Quickstart

Please refer the quickstart guides:

* [kind (local)](https://submariner.io/quickstart/kind/)
* [OpenShift (AWS)](https://submariner.io/quickstart/openshift/)
* [OpenShift with Globalnet (AWS)](https://submariner.io/quickstart/openshift/globalnet/)

## Subctl Releases

### Latest Stable Release

This release has the latest stable binaries: [latest release](https://github.com/submariner-io/submariner-operator/releases/latest)

### Latest Merged Release

This release is constantly updated with the latest code, and might be unstable: [devel
release](https://github.com/submariner-io/submariner-operator/releases/tag/devel)

## Building and Testing

See the [Building and Testing docs on Submainer's website](https://submariner.io/contributing/building_testing/).

## Reference

For reference, here's a link to the script generating the scaffold code of the 0.0.1 version of the operator
[gen_subm_operator.sh](https://github.com/submariner-io/submariner/blob/v0.0.2/operators/go/gen_subm_operator.sh).

## Updating OperatorHub

The OperatorHub definitions can be found here:

* [upstream-community-operators/submariner](https://github.com/operator-framework/community-operators/tree/master/upstream-community-operators/submariner)
* [community-operators/submariner](https://github.com/operator-framework/community-operators/tree/master/community-operators/submariner)
