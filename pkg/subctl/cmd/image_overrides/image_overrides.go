package image_overrides

import "github.com/spf13/cobra"

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
