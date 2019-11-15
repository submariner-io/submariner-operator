package scc

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/serviceaccount"
)

var (
	openshiftSCCGVR = schema.GroupVersionResource{
		Group:    "security.openshift.io",
		Version:  "v1",
		Resource: "securitycontextconstraints",
	}
)

func Ensure(restConfig *rest.Config, namespace string) (bool, error) {

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	sccClient := dynClient.Resource(openshiftSCCGVR)

	cr, err := sccClient.Get("privileged", metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	users, found, err := unstructured.NestedSlice(cr.Object, "users")
	if !found || err != nil {
		return false, err
	}

	submarinerUser := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, serviceaccount.OperatorServiceAccout)

	for _, user := range users {
		if submarinerUser == user.(string) {
			// the user is already part of the scc
			return false, nil
		}
	}

	if err = unstructured.SetNestedSlice(cr.Object, append(users, submarinerUser), "users"); err != nil {
		return false, err
	}

	if _, err = sccClient.Update(cr, metav1.UpdateOptions{}); err != nil {
		return false, fmt.Errorf("Error updating OpenShift privileged SCC: %s", err)
	}
	return true, nil
}
