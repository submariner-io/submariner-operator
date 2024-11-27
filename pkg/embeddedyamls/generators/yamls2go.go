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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var files = []string{
	"deploy/crds/submariner.io_brokers.yaml",
	"deploy/crds/submariner.io_submariners.yaml",
	"deploy/crds/submariner.io_servicediscoveries.yaml",
	"deploy/submariner/crds/submariner.io_clusters.yaml",
	"deploy/submariner/crds/submariner.io_endpoints.yaml",
	"deploy/submariner/crds/submariner.io_gateways.yaml",
	"deploy/submariner/crds/submariner.io_clusterglobalegressips.yaml",
	"deploy/submariner/crds/submariner.io_globalegressips.yaml",
	"deploy/submariner/crds/submariner.io_globalingressips.yaml",
	"deploy/submariner/crds/submariner.io_gatewayroutes.yaml",
	"deploy/submariner/crds/submariner.io_nongatewayroutes.yaml",
	"deploy/submariner/crds/submariner.io_routeagents.yaml",
	"deploy/mcsapi/crds/multicluster.x_k8s.io_serviceexports.yaml",
	"deploy/mcsapi/crds/multicluster.x_k8s.io_serviceimports.yaml",
	"config/broker/broker-admin/service_account.yaml",
	"config/broker/broker-admin/role.yaml",
	"config/broker/broker-admin/role_binding.yaml",
	"config/broker/broker-client/service_account.yaml",
	"config/broker/broker-client/role.yaml",
	"config/broker/broker-client/role_binding.yaml",
	"config/rbac/submariner-operator/service_account.yaml",
	"config/rbac/submariner-operator/role.yaml",
	"config/rbac/submariner-operator/role_binding.yaml",
	"config/rbac/submariner-operator/cluster_role.yaml",
	"config/rbac/submariner-operator/cluster_role_binding.yaml",
	"config/rbac/submariner-operator/ocp_cluster_role.yaml",
	"config/rbac/submariner-operator/ocp_cluster_role_binding.yaml",
	"config/rbac/submariner-gateway/service_account.yaml",
	"config/rbac/submariner-gateway/role.yaml",
	"config/rbac/submariner-gateway/role_binding.yaml",
	"config/rbac/submariner-gateway/cluster_role.yaml",
	"config/rbac/submariner-gateway/cluster_role_binding.yaml",
	"config/rbac/submariner-gateway/ocp_cluster_role.yaml",
	"config/rbac/submariner-gateway/ocp_cluster_role_binding.yaml",
	"config/rbac/submariner-route-agent/service_account.yaml",
	"config/rbac/submariner-route-agent/role.yaml",
	"config/rbac/submariner-route-agent/role_binding.yaml",
	"config/rbac/submariner-route-agent/cluster_role.yaml",
	"config/rbac/submariner-route-agent/cluster_role_binding.yaml",
	"config/rbac/submariner-route-agent/ocp_cluster_role.yaml",
	"config/rbac/submariner-route-agent/ocp_cluster_role_binding.yaml",
	"config/rbac/submariner-globalnet/service_account.yaml",
	"config/rbac/submariner-globalnet/role.yaml",
	"config/rbac/submariner-globalnet/role_binding.yaml",
	"config/rbac/submariner-globalnet/cluster_role.yaml",
	"config/rbac/submariner-globalnet/cluster_role_binding.yaml",
	"config/rbac/submariner-globalnet/ocp_cluster_role.yaml",
	"config/rbac/submariner-globalnet/ocp_cluster_role_binding.yaml",
	"config/rbac/submariner-diagnose/service_account.yaml",
	"config/rbac/submariner-diagnose/role.yaml",
	"config/rbac/submariner-diagnose/role_binding.yaml",
	"config/rbac/submariner-diagnose/cluster_role.yaml",
	"config/rbac/submariner-diagnose/cluster_role_binding.yaml",
	"config/rbac/submariner-diagnose/ocp_cluster_role.yaml",
	"config/rbac/submariner-diagnose/ocp_cluster_role_binding.yaml",
	"config/rbac/lighthouse-agent/service_account.yaml",
	"config/rbac/lighthouse-agent/cluster_role.yaml",
	"config/rbac/lighthouse-agent/cluster_role_binding.yaml",
	"config/rbac/lighthouse-agent/ocp_cluster_role.yaml",
	"config/rbac/lighthouse-agent/ocp_cluster_role_binding.yaml",
	"config/rbac/lighthouse-coredns/service_account.yaml",
	"config/rbac/lighthouse-coredns/cluster_role.yaml",
	"config/rbac/lighthouse-coredns/cluster_role_binding.yaml",
	"config/rbac/lighthouse-coredns/ocp_cluster_role.yaml",
	"config/rbac/lighthouse-coredns/ocp_cluster_role_binding.yaml",
	"config/openshift/rbac/submariner-metrics-reader/role.yaml",
	"config/openshift/rbac/submariner-metrics-reader/role_binding.yaml",
}

// Reads all .yaml files in the crdDirectory and encodes them as constants in yamls.go.
func main() {
	if len(os.Args) < 3 {
		fmt.Println("yamls2go needs two arguments, the base directory containing the YAML files, and the target directory")
		os.Exit(1)
	}

	yamlsDirectory := os.Args[1]
	goDirectory := os.Args[2]

	fmt.Println("Generating yamls.go")

	out, err := os.Create(filepath.Join(goDirectory, "yamls.go"))
	panicOnErr(err)

	_, err = out.WriteString(`/*
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

// This file is auto-generated by yamls2go.go
package embeddedyamls

const (
`)
	panicOnErr(err)

	// Raw string literals can’t contain backticks (which enclose the literals)
	// and there’s no way to escape them. Some YAML files we need to embed include
	// backticks... To work around this, without having to deal with all the
	// subtleties of wrapping arbitrary YAML in interpreted string literals, we
	// split raw string literals when we encounter backticks in the source YAML,
	// and add the backtick-enclosed string as an interpreted string:
	//
	// `resourceLock:
	//    description: The type of resource object that is used for locking
	//      during leader election. Supported options are ` + "`configmaps`" + ` (default)
	//      and ` + "`endpoints`" + `.
	//    type: string`

	re := regexp.MustCompile("`([^`]*)`")
	reNS := regexp.MustCompile(`(?s)\s*namespace:\s*placeholder\s*`)
	reDoc := regexp.MustCompile("(^|\n+)---\n")
	reTilde := regexp.MustCompile("`")

	for _, f := range files {
		_, err = out.WriteString("\t" + constName(f) + " = `")
		panicOnErr(err)

		fmt.Println(f)
		contents, err := os.ReadFile(path.Join(yamlsDirectory, f))
		panicOnErr(err)

		for _, index := range reDoc.FindAllIndex(contents, 2) {
			if index[0] > 0 {
				// Document starting inside the contents
				panic(f + " contains more than one document, use one file per document")
			}
		}

		_, err = out.Write(
			re.ReplaceAll(
				reNS.ReplaceAll(reTilde.ReplaceAll(contents, []byte("`"+"`")), []byte("\n")),
				[]byte("` + \"`$1`\" + `")))
		panicOnErr(err)

		_, err = out.WriteString("`\n")
		panicOnErr(err)
	}

	_, err = out.WriteString(")\n")
	panicOnErr(err)

	err = out.Close()
	panicOnErr(err)
}

func panicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func constName(filename string) string {
	return cases.Title(language.English).String(strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(filename,
				"-", "_"),
			".", "_"),
		"/", "_"))
}
