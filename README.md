# Submariner Operator

<!-- markdownlint-disable line-length -->
[![End to End Tests](https://github.com/submariner-io/submariner-operator/workflows/End%20to%20End%20Tests/badge.svg)](https://github.com/submariner-io/submariner-operator/actions?query=workflow%3A%22End+to+End+Tests%22)
[![Unit Tests](https://github.com/submariner-io/submariner-operator/workflows/Unit%20Tests/badge.svg)](https://github.com/submariner-io/submariner-operator/actions?query=workflow%3A%22Unit+Tests%22)
[![Linting](https://github.com/submariner-io/submariner-operator/workflows/Linting/badge.svg)](https://github.com/submariner-io/submariner-operator/actions?query=workflow%3ALinting)
[![Prometheus Tests](https://github.com/submariner-io/submariner-operator/workflows/Prometheus%20Tests/badge.svg)](https://github.com/submariner-io/submariner-operator/actions?query=workflow%3A%22Prometheus+Tests%22)
[![Release Images](https://github.com/submariner-io/submariner-operator/workflows/Release%20Images/badge.svg)](https://github.com/submariner-io/submariner-operator/actions?query=workflow%3A%22Release+Images%22)
[![Upgrade](https://github.com/submariner-io/submariner-operator/workflows/Upgrade/badge.svg)](https://github.com/submariner-io/submariner-operator/actions?query=workflow%3AUpgrade)
[![Periodic](https://github.com/submariner-io/submariner-operator/workflows/Periodic/badge.svg)](https://github.com/submariner-io/submariner-operator/actions?query=workflow%3APeriodic)
[![Flake Finder](https://github.com/submariner-io/submariner-operator/workflows/Flake%20Finder/badge.svg)](https://github.com/submariner-io/submariner-operator/actions?query=workflow%3A%22Flake+Finder%22)
<!-- markdownlint-enable line-length -->

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
