# Submariner Operator

The submariner operator installs the submariner components on a kubernetes cluster.

It's available on [OperatorHub:submariner](https://operatorhub.io/operator/submariner).

The current release is very basic, and will deploy the submariner components
in your cluster, but it does not deploy the broker yet.

# Working on the operator

You can compile the operator image running:
```bash
make build
```

The source code can be validated (golint, gofmt, unit testing) running:
```bash
make validate test
```

# Reference

For reference, here's a link to the script generating the scaffold code of the 0.0.1
version of the operator [gen_subm_operator.sh](https://github.com/submariner-io/submariner/blob/v0.0.2/operators/go/gen_subm_operator.sh).


# Updating OperatorHub

The OperatorHub definitions can be found here:
* https://github.com/operator-framework/community-operators/tree/master/upstream-community-operators/submariner
* https://github.com/operator-framework/community-operators/tree/master/community-operators/submariner

