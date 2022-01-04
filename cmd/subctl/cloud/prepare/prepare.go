package prepare

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/submariner-io/submariner-operator/internal/exit"
	"github.com/submariner-io/submariner-operator/internal/restconfig"
	"github.com/submariner-io/submariner-operator/pkg/cloud"
	"github.com/submariner-io/submariner-operator/pkg/cloud/prepare"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"os"
	"path/filepath"
)

var parentRestConfigProducer *restconfig.Producer

const (
	infraIDFlag = "infra-id"
	regionFlag  = "region"
	projectIDFlag = "project-id"
)

// NewCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func NewCommand(restConfigProducer *restconfig.Producer) *cobra.Command {
	parentRestConfigProducer = restConfigProducer
	cmd := &cobra.Command{
		Use:   "prepare",
		Short: "Prepare the cloud",
		Long:  `This command prepares the cloud for Submariner installation.`,
	}

	var port cloud.Ports
	cmd.PersistentFlags().Uint16Var(&port.Natt, "natt-port", 4500, "IPSec NAT traversal port")
	cmd.PersistentFlags().Uint16Var(&port.NatDiscovery, "nat-discovery-port", 4490, "NAT discovery port")
	cmd.PersistentFlags().Uint16Var(&port.Vxlan, "vxlan-port", 4800, "Internal VXLAN port")
	cmd.PersistentFlags().Uint16Var(&port.Metrics, "metrics-port", 8080, "Metrics port")

	cmd.AddCommand(newAWSPrepareCommand())
	cmd.AddCommand(newGCPPrepareCommand())
	cmd.AddCommand(newGenericPrepareCommand())

	return cmd
}

// newCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newAWSPrepareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "aws",
		Short: "Prepare an OpenShift AWS cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on AWS cloud for Submariner installation.",
		Run:   prepare.Aws,
	}

	var instance cloud.Instances
	addAWSFlags(cmd)
	cmd.Flags().StringVar(&instance.AWSGWType, "gateway-instance", "c5d.large", "Type of gateways instance machine")
	cmd.Flags().IntVar(&instance.Gateways, "gateways", cloud.DefaultNumGateways,
		"Number of dedicated gateways to deploy (Set to `0` when using --load-balancer mode)")

	return cmd
}

// newCommand returns a new cobra.Command used to prepare a cloud infrastructure.
func newGCPPrepareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gcp",
		Short: "Prepare an OpenShift GCP cloud",
		Long:  "This command prepares an OpenShift installer-provisioned infrastructure (IPI) on GCP cloud for Submariner installation.",
		Run:   prepare.GCP,
	}

	var instance cloud.Instances
	addGCPFlags(cmd)
	cmd.Flags().StringVar(&instance.GCPGWType, "gateway-instance", "n1-standard-4", "Type of gateway instance machine")
	cmd.Flags().IntVar(&instance.Gateways, "gateways", cloud.DefaultNumGateways,
		"Number of gateways to deploy")
	cmd.Flags().BoolVar(&instance.DedicatedGateway, "dedicated-gateway", false,
		"Whether a dedicated gateway node has to be deployed (default false)")

	return cmd
}

func newGenericPrepareCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generic",
		Short: "Prepares a generic cluster for Submariner",
		Long:  "This command labels the required number of gateway nodes for Submariner installation.",
		Run:   prepare.GenericCluster,
	}

	var instance cloud.Instances
	cmd.Flags().IntVar(&instance.Gateways, "gateways", cloud.DefaultNumGateways,
		"Number of gateways to deploy")

	return cmd
}

// addGCPFlags adds basic flags needed by GCP.
func addGCPFlags(command *cobra.Command) {
	var gcpOptions cloud.Options
	command.Flags().StringVar(&gcpOptions.InfraID, infraIDFlag, "", "GCP infra ID")
	command.Flags().StringVar(&gcpOptions.Region, regionFlag, "", "GCP region")
	command.Flags().StringVar(&gcpOptions.ProjectID, projectIDFlag, "", "GCP project ID")
	command.Flags().StringVar(&gcpOptions.OcpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or the directory containing it) from which to read the GCP infra ID "+
			"and region from (takes precedence over the specific flags)")

	dirname, err := os.UserHomeDir()
	if err != nil {
		utils.ExitOnError("failed to find home directory", err)
	}

	defaultCredentials := filepath.FromSlash(fmt.Sprintf("%s/.gcp/osServiceAccount.json", dirname))
	command.Flags().StringVar(&gcpOptions.CredentialsFile, "credentials", defaultCredentials, "GCP credentials configuration file")
}

// addAWSFlags adds basic flags needed by AWS.
func addAWSFlags(command *cobra.Command) {
	var awsOptions cloud.Options
	command.Flags().StringVar(&awsOptions.InfraID, infraIDFlag, "", "AWS infra ID")
	command.Flags().StringVar(&awsOptions.Region, regionFlag, "", "AWS region")
	command.Flags().StringVar(&awsOptions.OcpMetadataFile, "ocp-metadata", "",
		"OCP metadata.json file (or directory containing it) to read AWS infra ID and region from (Takes precedence over the flags)")
	command.Flags().StringVar(&awsOptions.Profile, "profile", "default", "AWS profile to use for credentials")

	dirname, err := os.UserHomeDir()
	if err != nil {
		exit.WithMessage(fmt.Sprintf("failed to find home directory", err))
	}

	defaultCredentials := filepath.FromSlash(fmt.Sprintf("%s/.aws/credentials", dirname))
	command.Flags().StringVar(&awsOptions.CredentialsFile, "credentials", defaultCredentials, "AWS credentials configuration file")
}