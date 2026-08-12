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

package cabledriver

import (
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	defaultIPsecHostPath = "/etc/ipsec.d"
	defaultNSSHostPath   = "/var/lib/ipsec/nss"
	defaultPlutoHostPath = "/var/run/pluto"

	ipsecHostPathOptionKey = "ipsecHostPath"
	nssHostPathOptionKey   = "ipsecNSSHostPath"
	plutoHostPathOptionKey = "plutoHostPath"
)

type libreswanDriver struct{}

func (libreswanDriver) Volumes(cr *v1alpha1.Submariner) ([]corev1.VolumeMount, []corev1.Volume) {
	if cr.Spec.CeIPSecUseOVNCertAuthMode {
		return libreswanCertModeVolumes(cr)
	}

	return libreswanPSKModeVolumes()
}

func libreswanPSKModeVolumes() ([]corev1.VolumeMount, []corev1.Volume) {
	return []corev1.VolumeMount{
			{Name: "ipsecd", MountPath: "/etc/ipsec.d", ReadOnly: false},
			{Name: "ipsecnss", MountPath: "/var/lib/ipsec/nss", ReadOnly: false},
		},
		[]corev1.Volume{
			{Name: "ipsecd", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
			{Name: "ipsecnss", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
		}
}

func libreswanCertModeVolumes(cr *v1alpha1.Submariner) ([]corev1.VolumeMount, []corev1.Volume) {
	hostPathType := corev1.HostPathDirectoryOrCreate
	ipsecHostPath := hostPathOption(cr, ipsecHostPathOptionKey, defaultIPsecHostPath)
	nssHostPath := hostPathOption(cr, nssHostPathOptionKey, defaultNSSHostPath)
	plutoHostPath := hostPathOption(cr, plutoHostPathOptionKey, defaultPlutoHostPath)

	return []corev1.VolumeMount{
			{Name: "ipsecd", MountPath: "/etc/ipsec.d", ReadOnly: false},
			{Name: "ipsecnss", MountPath: "/var/lib/ipsec/nss", ReadOnly: false},
			{Name: "plutosocket", MountPath: "/var/run/pluto", ReadOnly: false},
		},
		[]corev1.Volume{
			{Name: "ipsecd", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
				Path: ipsecHostPath, Type: &hostPathType,
			}}},
			{Name: "ipsecnss", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
				Path: nssHostPath, Type: &hostPathType,
			}}},
			{Name: "plutosocket", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{
				Path: plutoHostPath, Type: &hostPathType,
			}}},
		}
}

func hostPathOption(cr *v1alpha1.Submariner, key, defaultPath string) string {
	if path := cr.Spec.CableDriverOptions[key]; path != "" {
		return path
	}

	return defaultPath
}
