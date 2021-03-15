/*
© 2019 Red Hat, Inc. and others.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package embeddedyamls

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type IObject struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func GetObject(yamlStr string, obj interface{}) error {
	doc := []byte(yamlStr)

	if err := yaml.Unmarshal(doc, obj); err != nil {
		return err
	}

	return nil
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
