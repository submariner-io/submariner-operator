package cloud

type Options struct {
	InfraID         string
	Region          string
	Profile         string
	CredentialsFile string
	OcpMetadataFile string
	ProjectID       string
}

type Ports struct {
	Natt         uint16
	NatDiscovery uint16
	Vxlan        uint16
	Metrics      uint16
}

type Instances struct {
	AWSGWType string
	GCPGWType string
	Gateways          int
	DedicatedGateway  bool
}

const DefaultNumGateways = 1
