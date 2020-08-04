# Submariner Operator

The submariner operator installs the submariner components on a Kubernetes cluster.

It's available on [OperatorHub:submariner](https://operatorhub.io/operator/submariner).

## Quickstart

Please refer the quickstart guides:

* [OpenShift AWS](https://submariner-io.github.io/quickstart/openshift)
* [OpenShift AWS With Service Discovery](https://submariner-io.github.io/quickstart/openshiftsd)
* [OpenShift With Service Discovery And GlobalNet](https://submariner-io.github.io/quickstart/openshiftgn/)

## Subctl Releases

### Latest Stable Release

This release has the latest stable binaries:
[latest release](https://github.com/submariner-io/submariner-operator/releases/latest)

### Latest Merged Release

This release is constantly updated with the latest code, and might be unstable:
[devel release](https://github.com/submariner-io/submariner-operator/releases/tag/devel)

## Development

### Prerequisites

The build environment uses
[Dapper](https://github.com/rancher/dapper), which requires a working
Docker setup. You will also need GNU Make.

### Build Operator

 You can compile the operator image running:

```shell
make build-operator
```

The source code can be validated (golint, gofmt, unit testing) running:

```shell
make validate test
```

### Build Subctl

To build subctl locally

```shell
go mod vendor
make bin/subctl
```

You will be able to run subctl using ./bin/subctl from the submariner-operator directory.

### Testing

To run end-to-end tests:

```shell
make e2e cleanup
```

### Setup development environment

You will need docker installed in your system, and at least 8GB of RAM. Run:

```shell
make deploy
```

## Reference

For reference, here's a link to the script generating the scaffold code of the 0.0.1
version of the operator [gen_subm_operator.sh](https://github.com/submariner-io/submariner/blob/v0.0.2/operators/go/gen_subm_operator.sh).

## Updating OperatorHub

The OperatorHub definitions can be found here:

* [upstream-community-operators/submariner](https://github.com/operator-framework/community-operators/tree/master/upstream-community-operators/submariner)
* [community-operators/submariner](https://github.com/operator-framework/community-operators/tree/master/community-operators/submariner)
