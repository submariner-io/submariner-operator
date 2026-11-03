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

// Package cabledriver customizes gateway DaemonSet construction per cable driver.
//
// To add gateway volumes (or other DaemonSet bits) for a new cable driver:
//  1. Add <driver>.go in this package implementing Driver
//  2. Register it in init() via AddDriver (see libreswan.go)
//
// Drivers with no operator-side volumes stay unregistered and contribute nothing.
package cabledriver

import (
	"github.com/submariner-io/admiral/pkg/log"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// Driver customizes gateway DaemonSet construction for a cable driver.
type Driver interface {
	Volumes(cr *v1alpha1.Submariner) ([]corev1.VolumeMount, []corev1.Volume)
}

var (
	drivers = map[string]Driver{}
	logger  = log.Logger{Logger: logf.Log.WithName("CableDriver")}
)

// AddDriver registers a cable driver for gateway DaemonSet customization.
func AddDriver(name string, driver Driver) {
	if drivers[name] != nil {
		logger.Fatalf("Multiple gateway cable drivers attempting to register with name %q", name)
	}

	drivers[name] = driver
}

// Volumes returns cable-driver-specific volume mounts and volumes for the gateway pod.
func Volumes(cr *v1alpha1.Submariner) ([]corev1.VolumeMount, []corev1.Volume) {
	if driver := drivers[cr.Spec.CableDriver]; driver != nil {
		return driver.Volumes(cr)
	}

	return nil, nil
}
