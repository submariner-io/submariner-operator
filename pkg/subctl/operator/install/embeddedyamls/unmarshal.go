package embeddedyamls

import "sigs.k8s.io/yaml"

func GetObject(yamlStr string, obj interface{}) error {

	doc := []byte(yamlStr)

	if err := yaml.Unmarshal(doc, obj); err != nil {
		return err
	}

	return nil
}
