/*
© 2021 Red Hat, Inc. and others.

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
package gather

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

func ResourcesToYAMLFile(info *Info, ofType schema.GroupVersionResource, namespace string) {
	err := func() error {
		list, err := info.DynClient.Resource(ofType).Namespace(namespace).List(metav1.ListOptions{})
		if err != nil {
			return errors.WithMessagef(err, "error listing %q", ofType.Resource)
		}

		path := filepath.Join(info.DirName, info.ClusterName+"-"+ofType.Resource+".yaml")
		file, err := os.Create(path)
		if err != nil {
			return errors.WithMessagef(err, "error opening file %s", path)
		}

		defer file.Close()

		data, err := yaml.Marshal(list)
		if err != nil {
			return errors.WithMessage(err, "error marshaling to YAML")
		}

		_, err = file.Write(data)
		if err != nil {
			return errors.WithMessagef(err, "error writing to file %s", path)
		}

		return nil
	}()

	if err != nil {
		info.Status.QueueFailureMessage(fmt.Sprintf("Failed to gather %s: %s", ofType.Resource, err))
	} else {
		info.Status.QueueSuccessMessage(fmt.Sprintf("Successfully gathered %s", ofType.Resource))
	}
}
