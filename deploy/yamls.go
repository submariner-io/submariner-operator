/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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

package deploy

import (
	"embed"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

//go:embed crds mcsapi submariner
var Yamls embed.FS

type IObject struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

func GetObject(yamlFile string, obj interface{}) error {
	yamlObj, err := Yamls.ReadFile(yamlFile)
	if err != nil {
		return errors.Wrapf(err, "error reading yaml file %s", yamlFile)
	}

	if err := yaml.Unmarshal(yamlObj, obj); err != nil {
		return errors.Wrapf(err, "error unmarshalling object")
	}

	return nil
}

func GetObjectName(yamlFile string) (string, error) {
	var obj IObject

	yamlobj, err := Yamls.ReadFile(yamlFile)
	if err != nil {
		return "", errors.Wrapf(err, "error reading yaml %s", yamlFile)
	}

	err = yaml.Unmarshal(yamlobj, &obj)
	if err != nil {
		return "", errors.Wrapf(err, "error unmarshalling object")
	}

	return obj.Name, nil
}
