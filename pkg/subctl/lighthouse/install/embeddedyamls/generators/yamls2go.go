/*
© 2020 Red Hat, Inc. and others.

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
	"io"
	"os"
	"path"
	"strings"
)

const (
	yamlsDirectory = "../../../../../deploy/lighthouse"
)

var files = []string{
	"crds/multiclusterservices_crd.yaml",
}

// Reads all .yaml files in the crdDirectory
// and encodes them as constants in crdyamls.go
func main() {

	fmt.Println("Generating yamls.go")
	out, err := os.Create("yamls.go")
	panicOnErr(err)

	_, err = out.WriteString("// This file is auto-generated by yamls2go.go\n" +
		"package embeddedyamls\n\nconst (\n")
	panicOnErr(err)

	for _, f := range files {

		_, err = out.WriteString("\t" + constName(f) + " = `")
		panicOnErr(err)

		fmt.Println(f)
		f, _ := os.Open(path.Join(yamlsDirectory, f))
		_, err = io.Copy(out, f)
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
	return strings.Title(strings.Replace(
		strings.Replace(filename,
			".", "_", -1),
		"/", "_", -1))
}
