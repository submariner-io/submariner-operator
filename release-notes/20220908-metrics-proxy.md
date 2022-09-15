<!-- markdownlint-disable MD041 -->
Users no longer need to open ports `8080` and `8081` on the host for querying metrics. A new `submariner-metrics-proxy`
DaemonSet runs pods on gateway nodes and forwards http requests for metrics services to gateway and globalnet pods running
on the nodes. Gateway and Globalnet pods now listen on ports `32780` and `32781` instead of well known ports `8080` and
`8081` to avoid conflict with any other services that might be using those ports. Users will continue to query existing
`submariner-gateway-metrics` and `submariner-globalnet-metrics` services to query the metrics.
