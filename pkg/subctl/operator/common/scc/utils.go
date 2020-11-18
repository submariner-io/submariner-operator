package scc

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

var (
	openshiftSCCGVR = schema.GroupVersionResource{
		Group:    "security.openshift.io",
		Version:  "v1",
		Resource: "securitycontextconstraints",
	}
)

func UpdateSCC(restConfig *rest.Config, namespace, name string) (bool, error) {
	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	sccClient := dynClient.Resource(openshiftSCCGVR)

	created := false
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		cr, err := sccClient.Get("privileged", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		users, found, err := unstructured.NestedSlice(cr.Object, "users")
		if !found || err != nil {
			return err
		}

		submarinerUser := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, name)

		for _, user := range users {
			if submarinerUser == user.(string) {
				// the user is already part of the scc
				return nil
			}
		}

		if err := unstructured.SetNestedSlice(cr.Object, append(users, submarinerUser), "users"); err != nil {
			return err
		}

		if _, err = sccClient.Update(cr, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("Error updating OpenShift privileged SCC: %s", err)
		}
		created = true
		return nil
	})
	return created, retryErr
}
