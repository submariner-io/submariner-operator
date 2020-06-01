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

package lighthousedns

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

func Ensure(status *cli.Status, config *rest.Config, repo string, version string) error {
	clientSet, err := clientset.NewForConfig(config)
	if err != nil {
		return err
	}

	// Set up the CoreDNS cluster role (if present)
	err = setupClusterRole(status, clientSet, "system:coredns", "multiclusterservices", true)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	// Set up the OpenShift DNS cluster role (if present)
	err = setupClusterRole(status, clientSet, "openshift-dns", "multiclusterservices", false)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}

func setupClusterRole(status *cli.Status, clientSet *clientset.Clientset, name string, resources string, cud bool) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		clusterRole, err := clientSet.RbacV1().ClusterRoles().Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		verbs := []string{"list", "watch", "get"}
		if cud {
			verbs = append(verbs, "create", "delete", "update")
		}
		if clusterRole != nil {
			apiGroupSeen := false
			for _, rule := range clusterRole.Rules {
				for _, apiGroup := range rule.APIGroups {
					if apiGroup == "lighthouse.submariner.io" {
						apiGroupSeen = true
						rule.Resources = []string{resources}
						rule.Verbs = verbs
						status.QueueSuccessMessage("Updated existing Lighthouse entry in the " + name + " role")
					}
				}
			}
			if !apiGroupSeen {
				rule := rbacv1.PolicyRule{}
				rule.APIGroups = []string{"lighthouse.submariner.io"}
				rule.Resources = []string{resources}
				rule.Verbs = verbs
				clusterRole.Rules = append(clusterRole.Rules, rule)
				status.QueueSuccessMessage("Added Lighthouse entry in the " + name + " role")
			}
			// Potentially retried
			_, err = clientSet.RbacV1().ClusterRoles().Update(clusterRole)
			return err
		}
		return nil
	})
	// “Is not found” errors are returned as-is
	if retryErr != nil && !errors.IsNotFound(retryErr) {
		return fmt.Errorf("Error setting up the %s cluster role: %v", name, retryErr)
	}
	return retryErr
}
