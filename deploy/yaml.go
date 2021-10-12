package deploy

import (
	"embed"
	"fmt"
)

//go:embed mcsapi crds submariner
var DeployYamls embed.FS

func GetEmbeddedYaml(filename string) string {
	crdYaml, err := DeployYamls.ReadFile(filename)
	if err != nil {
		fmt.Println("error accesing CRD Yaml file", filename)
	}
	return string(crdYaml)
}
