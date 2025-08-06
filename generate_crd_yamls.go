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

package main

import (
	"os"
	"path/filepath"
	"strings"

	opcrds "github.com/submariner-io/submariner-operator/deploy/crds"
	submcrds "github.com/submariner-io/submariner/deploy/crds"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	msccrd "sigs.k8s.io/mcs-api/config/crd"
)

// This program is used by submariner-charts to generate CRD yaml files from embedded strings referenced
// from Go dependencies.
func main() {
	yamlsDirectory := os.Args[1]

	writeYamlFile(opcrds.BrokerCRD, yamlsDirectory, "brokers.yaml")
	writeYamlFile(opcrds.SubmarinerCRD, yamlsDirectory, "submariners.yaml")
	writeYamlFile(opcrds.ServiceDiscoveryCRD, yamlsDirectory, "servicediscoveries.yaml")

	writeYamlFile(submcrds.EndpointsCRD, yamlsDirectory, "endpoints.yaml")
	writeYamlFile(submcrds.ClustersCRD, yamlsDirectory, "clusters.yaml")
	writeYamlFile(submcrds.GatewaysCRD, yamlsDirectory, "gateways.yaml")

	writeYamlFile(msccrd.ServiceExportCRD, yamlsDirectory, "serviceexports.yaml")
	writeYamlFile(msccrd.ServiceImportCRD, yamlsDirectory, "serviceimports.yaml")
}

func writeYamlFile(b []byte, dir, fileName string) {
	out, err := os.Create(filepath.Join(dir, fileName))
	utilruntime.Must(err)

	defer out.Close()

	yaml := string(b)

	if !strings.HasPrefix(yaml, "---") {
		_, err = out.WriteString("---\n")
		utilruntime.Must(err)
	}

	_, err = out.WriteString(yaml)
	utilruntime.Must(err)
}
