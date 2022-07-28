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

package controllers

import (
	operatorclient "github.com/openshift/cluster-dns-operator/pkg/operator/client"
	"github.com/submariner-io/submariner-operator/controllers/servicediscovery"
	"github.com/submariner-io/submariner-operator/controllers/submariner"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManager adds all Controllers to the Manager.
// nolint:wrapcheck // No need to wrap errors here.
func AddToManager(mgr manager.Manager) error {
	kubeClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	generalClient, _ := operatorclient.NewClient(mgr.GetConfig())

	if err := submariner.NewReconciler(&submariner.Config{
		ScopedClient:  mgr.GetClient(),
		GeneralClient: generalClient,
		RestConfig:    mgr.GetConfig(),
		Scheme:        mgr.GetScheme(),
		DynClient:     dynamic.NewForConfigOrDie(mgr.GetConfig()),
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return servicediscovery.NewReconciler(&servicediscovery.Config{
		Client:     mgr.GetClient(),
		RestConfig: mgr.GetConfig(),
		Scheme:     mgr.GetScheme(),
		KubeClient: kubeClient,
	}).SetupWithManager(mgr)
}
