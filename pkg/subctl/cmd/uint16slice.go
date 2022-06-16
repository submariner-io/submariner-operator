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

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Uint16Slice struct {
	Value *[]uint16
}

func (s *Uint16Slice) Type() string {
	return "Uint16Slice"
}

func (s *Uint16Slice) String() string {
	return fmt.Sprintf("%v", *s.Value)
}

func (s *Uint16Slice) Set(value string) error {
	values := strings.Split(value, ",")

	*s.Value = make([]uint16, len(values))

	for i, d := range values {
		u, err := strconv.ParseUint(d, 10, 16)
		if err != nil {
			return errors.Wrap(err, "conversion to uint16 failed")
		}

		(*s.Value)[i] = uint16(u)
	}

	return nil
}
