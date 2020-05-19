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
	"time"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	clientsetifc "k8s.io/client-go/kubernetes/typed/apps/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

const (
	operatorImage           = "lighthouse-cluster-dns-operator"
	openShiftCoreDNSImage   = "openshift-lighthouse-coredns"
	deploymentCheckInterval = 5 * time.Second
	deploymentWaitTime      = 10 * time.Minute
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

	// Disable ClusterVersionOperator on OpenShift
	// CVO keeps track of OpenShift's required services, which include DNS;
	// it undoes any change we make to DNS. Disabling CVO allows us to make
	// our changes and keep them, but it means the cluster no longer upgrades
	// itself (so this is not supportable)
	cvoReplicas, err := scaleDeployment(clientSet.AppsV1().Deployments("openshift-cluster-version"), "cluster-version-operator", 0)
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	if err == nil {
		// We're on OpenShift
		if cvoReplicas > 0 {
			status.QueueSuccessMessage("Disabled the cluster version operator")
		}
		err = setupOpenShift(status, clientSet, repo, version)
		if err != nil {
			return err
		}
	} else {
		return nil
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

func setupOpenShift(status *cli.Status, clientSet *clientset.Clientset, repo string, version string) error {
	// Fix up cluster role
	err := setupClusterRole(status, clientSet, "openshift-dns-operator", "multicluster", false)
	if err != nil {
		return err
	}

	// Update operator deployment
	deployments := clientSet.AppsV1().Deployments("openshift-dns-operator")
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		deployment, err := deployments.Get("dns-operator", metav1.GetOptions{})
		if errors.IsNotFound(err) {
			// Assume we're not on OpenShift
			return nil
		} else if err != nil {
			return err
		}
		for i, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == "dns-operator" {
				deployment.Spec.Template.Spec.Containers[i].Image = repo + operatorImage + ":" + version
				for j, env := range container.Env {
					if env.Name == "IMAGE" {
						deployment.Spec.Template.Spec.Containers[i].Env[j].Value = repo + openShiftCoreDNSImage + ":" + version
					}
				}
				status.QueueSuccessMessage("Updated DNS operator deployment")
			}
		}
		// Potentially retried
		_, err = deployments.Update(deployment)
		return err
	})
	if retryErr != nil {
		return fmt.Errorf("Error updating dns-operator deployment: %v", retryErr)
	}

	// Scale DNS operator down and back up
	originalReplicas, err := scaleDeployment(deployments, "dns-operator", 0)
	if err != nil {
		if errors.IsNotFound(err) {
			// Assume we're not on OpenShift
			return nil
		}
		return err
	}
	_, err = scaleDeployment(deployments, "dns-operator", originalReplicas)
	if err != nil {
		if errors.IsNotFound(err) {
			// Assume we're not on OpenShift
			return nil
		}
		return err
	}
	status.QueueSuccessMessage("Restarted the DNS operator")

	return nil
}

func scaleDeployment(deployments clientsetifc.DeploymentInterface, deploymentName string, targetReplicas int32) (int32, error) {
	scale, err := deployments.GetScale(deploymentName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	originalReplicas := scale.Spec.Replicas
	scale.Spec.Replicas = targetReplicas
	_, err = deployments.UpdateScale(deploymentName, scale)
	if err != nil {
		return originalReplicas, err
	}
	err = waitForReplicas(deployments, deploymentName, targetReplicas)
	return originalReplicas, err
}

func waitForReplicas(deployments clientsetifc.DeploymentInterface, deploymentName string, targetReplicas int32) error {
	return wait.PollImmediate(deploymentCheckInterval, deploymentWaitTime, func() (bool, error) {
		check, err := deployments.Get(deploymentName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("error waiting for replicas to adjust: %s", err)
		}

		return check.Status.Replicas == targetReplicas, nil
	})
}
