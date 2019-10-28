# E2E environment

The e2e environment can be used for local testing, development, and CI purposes.

E2E environment consists of:
- 3 k8s clusters deployed with [kind].
  - Cluster1: One master node for broker and elasticsearch if required.
  - Cluster{2..3}: One master node and two workers for gateways nodes.
  The configuration can be viewed or changed at **scripts/kind-e2e/cluster{1..3}-config.yaml**.
- Submariner installed and configured on top of the clusters.

[kind] is a tool for running local Kubernetes clusters using Docker container “nodes”.

## Prerequisites

- [docker]
- [kubectl]
- [Add local user to docker group]

Optional useful tools for troubleshooting:

- [k9s]
- [kubetail]

## Installation and usage

The environment can be used in two modes:

### 1. Ephemeral/Onetime
 
This mode will create e2e environment on a local workstation, run unit tests, E2E tests, and clean up the created resources afterwards.
This mode is very convenient for CI purposes.
    
```bash
make ci e2e
```

If you want to minimize runtime another option is to skip unit tests and source
code validation:
```bash
make build package e2e
```

To test specific k8s version, additional **version** parameter can be passed to **make e2e** command.

```bash
make ci e2e version=1.14.1
```

Full list of supported k8s versions can found on [kind release page] page. We are using kind vesion 0.3.0.
Default **version** is 1.14.2.

### 2. Permanent

This mode will **keep** the e2e environment running on a local workstation for further development and debugging.

This mode can be triggered by adding **status=keep** parameter to **make e2e** command.

```bash
make ci e2e status=keep
```

After a permanent run completes, the configuration for the running clusters can be found inside **output/kind-config/local-dev** folder.
You can export the kube configs in order to interact with the clusters.

```bash
export KUBECONFIG=$(echo $(git rev-parse --show-toplevel)/output/kind-config/local-dev/kind-config-cluster{1..3} | sed 's/ /:/g')
```

List the contexts:

```bash
kubectl config list-contexts
```

You should be able to see 3 contexts. From this stage you can interact with the clusters
as with any normal k8s cluster.

**NOTE**: Each time **make e2e status=keep** command is executed, the local code will be build, pushed to kind clusters
as docker images, submariner will be redeployed on the clusters from pushed images and E2E tests will be executed.
This mode allows the developers to test their local code fast on a very close to real world scenario setup.

#### Cleanup
At any time you can run a cleanup command that will remove kind resources.

```bash
make ci e2e status=clean
```

You can do full docker cleanup, but it will force all the docker images to be removed and invalidate the local docker cache. 
The next run will be a cold run and will take more time.

```bash
docker system prune --all
``` 

#### Full example

```bash
make ci e2e status=keep version=1.14.1
```

<!--links-->
[kind]: https://github.com/kubernetes-sigs/kind
[docker]: https://docs.docker.com/install/
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[k9s]: https://github.com/derailed/k9s
[kubetail]: https://github.com/johanhaleby/kubetail
[kind release page]: https://github.com/kubernetes-sigs/kind/releases
[Add local user to docker group]: https://docs.docker.com/install/linux/linux-postinstall/
