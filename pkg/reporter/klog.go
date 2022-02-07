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

import "k8s.io/klog/v2"

type klogType struct{}

func Klog() Interface {
	return &klogType{}
}

func (k klogType) Start(message string, args ...interface{}) {
	klog.Infof(message, args...)
}

func (k klogType) End() {
}

func (k klogType) Success(message string, args ...interface{}) {
	klog.Infof(message, args...)
}

func (k klogType) Failure(message string, args ...interface{}) {
	klog.Errorf(message, args...)
}

func (k klogType) Warning(message string, args ...interface{}) {
	klog.Warningf(message, args...)
}

func (k klogType) Error(err error, message string, args ...interface{}) error {
	return HandleError(k, err, message, args...)
}
