package names

const (
	RouteAgentImage        = "submariner-route-agent"
	EngineImage            = "submariner"
	GlobalnetImage         = "submariner-globalnet"
	ServiceDiscoveryImage  = "lighthouse-agent"
	LighthouseCoreDNSImage = "lighthouse-coredns"
	ServiceDiscoveryCrName = "service-discovery"
)

var (
	// ImagePrefix is used by downstream distributions to introduce a prefix in the component
	ImagePrefix = ""
	// ImagePostfix is used by downstream distributions to introduce a postfix in the component image
	ImagePostfix = ""
)
