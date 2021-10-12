package config

import (
	"embed"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

//go:embed broker bundle certmanager crd default manager manifests openshift prometheus rbac samples scorecard webhook
var ConfigYamls embed.FS

type IObject struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func GetEmbeddedYaml(filename string) string {
	crdYaml, err := ConfigYamls.ReadFile(filename)
	if err != nil {
		fmt.Println("error accesing CRD Yaml file", filename)
	}
	return string(crdYaml)
}

func GetObjectName(yamlStr string) (string, error) {
	doc := []byte(yamlStr)
	var obj IObject

	err := yaml.Unmarshal(doc, &obj)
	if err != nil {
		return "", err
	}

	return obj.Name, err
}

func GetObject(yamlStr string, obj interface{}) error {
	doc := []byte(yamlStr)

	if err := yaml.Unmarshal(doc, obj); err != nil {
		return err
	}

	return nil
}