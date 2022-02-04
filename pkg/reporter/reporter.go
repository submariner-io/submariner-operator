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

package reporter

import (
	"unicode"

	"github.com/pkg/errors"
)

// Interface for reporting on the progress of an operation.
type Interface interface {
	// Start reports that an operation or sequence of operations is starting.
	Start(message string, args ...interface{})

	// Success reports that the last operation succeeded with the specified message.
	Success(message string, args ...interface{})

	// Failure reports that the last operation failed with the specified message.
	Failure(message string, args ...interface{})

	// End the current operation that was previously initiated via Start.
	End()

	// Warning reports a warning message for the last operation.
	Warning(message string, args ...interface{})

	// Error wraps err with the supplied message, reports it as a failure, ends the current operation, and returns the error.
	Error(err error, message string, args ...interface{}) error
}

func HandleError(reporter Interface, err error, message string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	err = errors.Wrapf(err, message, args...)

	capitalizeFirst := func(str string) string {
		for i, v := range str {
			return string(unicode.ToUpper(v)) + str[i+1:]
		}

		return ""
	}

	reporter.Failure(capitalizeFirst(err.Error()))
	reporter.End()

	return err
}
