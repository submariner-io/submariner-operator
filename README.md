# Submariner Operator

The submariner operator installs the submariner components on a Kubernetes cluster.

It's available on [OperatorHub:submariner](https://operatorhub.io/operator/submariner).

# Quickstart

## Prerequisites

Submariner has a few requirements to get started:

- At least 2 Kubernetes clusters, one of which is designated to serve as the central broker that is accessible by all of your connected clusters; this can be one of your connected clusters, but comes with the limitation that the cluster is required to be up to facilitate interconnectivity/negotiation
- Different cluster/service CIDR's (as well as different Kubernetes DNS suffixes) between clusters. This is to prevent traffic selector/policy/routing conflicts.
- Direct IP connectivity between the gateway nodes through the internet (or on the same network if not running Submariner over the internet). Submariner supports 1:1 NAT setups but has a few caveats/provider-specific configuration instructions in this configuration.
- Knowledge of each cluster's network configuration
- Worker node IPs on all the clusters must be outside of the cluster/service CIDR ranges.

An example of three clusters configured to use with Submariner would look like the following:

| Cluster Name | Provider | Cluster CIDR | Service CIDR | DNS Suffix |
|:-------------|:---------|:-------------|:-------------|:-----------|
| broker       | AWS      | 10.42.0.0/16 | 10.43.0.0/16 | cluster.local |
| west         | vSphere  | 10.0.0.0/16  | 10.1.0.0/16  | west.local |
| east         | OnPrem   | 10.98.0.0/16 | 10.99.0.0/16 | east.local |

## Installation

### Download
Download the latest binary for your operating system and architecture, then install it on your path. The binary has no external dependencies.

To install for Linux amd64 in `~/.local/bin`:

    mkdir -p ~/.local/bin
    wget https://github.com/submariner-io/submariner-operator/releases/download/v0.1.0/subctl-v0.1.0-linux-amd64
    install subctl-v0.1.0 -linux-amd64 ~/.local/bin/subctl
    rm subctl-v0.1.0-linux-amd64
If `~/.local/bin` is on your PATH, you will then be able to run subctl.

## Deployment

### Submariner

To deploy submariner in a two Kubernetes cluster setup follow the below steps

* Deploy Broker with Dataplane - To deploy the Submariner broker along with the data cluster run
```
subctl deploy-broker --kubeconfig <PATH-TO-KUBECONFIG-BROKER> --dataplane
```


* Join Clusters with Broker - To join all the other clusters with the broker cluster, run using the broker-info.subm generated in the folder from which the previous step was run.

```
subctl join --kubeconfig <PATH-TO-KUBECONFIG-DATA-CLUSTER> broker-info.subm
```

### Submariner with Service Discovery

 **Project status**: The project is meant only to be used as a development preview. Installing the operator on an Openshift cluster may disable some of the operator features.

The [Lighthouse](https://github.com/submariner-io/lighthouse) helps in cross-cluster service discovery. It has the below additional dependencies

- kubefedctl installed ([0.1.0-rc3](https://github.com/kubernetes-sigs/kubefed/releases/tag/v0.1.0-rc3)).
- kubectl installed.

To deploy Submariner with Lighthouse follow the below steps.

* Deploy Broker with Dataplane - To deploy the Submariner broker along with the data cluster, run using the broker-info.subm generated in the folder from which the previous step was run.
```
subctl deploy-broker --kubeconfig <PATH-TO-KUBECONFIG-BROKER> --dataplane --service-discovery --broker-cluster-context <BROKER-CONTEXT-NAME>
```

* Join Clusters with Broker - To join all the other clusters with the broker cluster run using the broker-info.subm generated in the folder from which the previous step was run.

```
subctl join --kubeconfig <PATH-TO-KUBECONFIG-DATA-CLUSTER> broker-info.subm  --broker-cluster-context <BROKER-CONTEXT-NAME>
```


# Development
 
## Build Operator
 
 You can compile the operator image running:
```bash
make build
```

The source code can be validated (golint, gofmt, unit testing) running:
```bash
make validate test
```

## Build Subctl

To build subctl locally
```bash
make bin/subctl
```
You will be able to run subctl using ./bin/subctl from the submariner-operator directory.
 
## Testing
To run end-to-end tests:
```
make e2e
```
 
## Setup development environment 
You will need docker installed in your system, and at least 8GB of RAM. Run:

 ```
 make e2e status=keep
 ```
 
 

# Reference

For reference, here's a link to the script generating the scaffold code of the 0.0.1
version of the operator [gen_subm_operator.sh](https://github.com/submariner-io/submariner/blob/v0.0.2/operators/go/gen_subm_operator.sh).


# Updating OperatorHub

The OperatorHub definitions can be found here:
* https://github.com/operator-framework/community-operators/tree/master/upstream-community-operators/submariner
* https://github.com/operator-framework/community-operators/tree/master/community-operators/submariner

