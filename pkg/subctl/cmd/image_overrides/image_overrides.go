package image_overrides

import (
	"github.com/spf13/cobra"

	"github.com/submariner-io/submariner-operator/pkg/versions"
)

type VersionRegistryOverrides struct {
	version  string
	registry string
}
type ImageOverrides struct {
	operator   VersionRegistryOverrides
	submariner VersionRegistryOverrides
	lighthouse VersionRegistryOverrides
}

func (img *ImageOverrides) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&img.operator.registry, "operator-registry", "",
		"Operator container registry")
	cmd.Flags().StringVar(&img.operator.version, "operator-version", "",
		"Operator container version")

	cmd.Flags().StringVar(&img.submariner.registry, "submariner-registry", "",
		"Submariner container registry")
	cmd.Flags().StringVar(&img.submariner.version, "submariner-version", "",
		"Submariner container version")

	cmd.Flags().StringVar(&img.lighthouse.registry, "lighthouse-registry", "",
		"Lighthouse container registry")
	cmd.Flags().StringVar(&img.lighthouse.version, "lighthouse-version", "",
		"Lighthouse container version")

	_ = cmd.Flags().MarkHidden("operator-registry")
	_ = cmd.Flags().MarkHidden("operator-version")
	_ = cmd.Flags().MarkHidden("submariner-registry")
	_ = cmd.Flags().MarkHidden("submariner-version")
	_ = cmd.Flags().MarkHidden("lighthouse-registry")
	_ = cmd.Flags().MarkHidden("lighthouse-version")
}
func defaultStr(defaultVal, override string) string {
	if override != "" {
		return override
	} else {
		return defaultVal
	}
}

func (img *ImageOverrides) GetSubmarinerRegistryAndVersion(defaultRegistry, defaultVersion string) (registry, version string) {
	defaultVersion = defaultStr(defaultVersion, versions.DefaultSubmarinerVersion)
	registry = defaultStr(defaultRegistry, img.submariner.registry)
	version = defaultStr(defaultVersion, img.submariner.version)

	return registry, version
}

func (img *ImageOverrides) GetLighthouseRegistryAndVersion(defaultRegistry, defaultVersion string) (registry, version string) {
	defaultVersion = defaultStr(defaultVersion, versions.DefaultLighthouseVersion)
	registry = defaultStr(defaultRegistry, img.submariner.registry)
	version = defaultStr(defaultVersion, img.submariner.version)

	return registry, version
}

func (img *ImageOverrides) IsSubmarinerLighthouseUnmatched() bool {
	return img.submariner.registry != img.lighthouse.registry ||
		img.submariner.version != img.lighthouse.version
}

func (img *ImageOverrides) GetOperatorRegistryAndVersion(defaultRegistry, defaultVersion string) (registry, version string) {
	defaultVersion = defaultStr(defaultVersion, versions.DefaultSubmarinerOperatorVersion)
	registry = defaultStr(defaultRegistry, img.operator.registry)
	version = defaultStr(defaultVersion, img.operator.version)

	return registry, version
}
