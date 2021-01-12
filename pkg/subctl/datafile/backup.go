/*
© 2021 Red Hat, Inc. and others

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
package datafile

import (
	"os"
	"strings"
	"time"
)

func BackupIfExists(filename string) (string, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return "", nil
	}
	now := time.Now()
	nowStr := strings.ReplaceAll(now.Format(time.RFC3339), ":", "_")
	newFilename := filename + "." + nowStr
	return newFilename, os.Rename(filename, newFilename)
}
