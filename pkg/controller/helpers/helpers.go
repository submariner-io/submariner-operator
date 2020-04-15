package helpers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	errorutil "github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func ReconcileDaemonSet(owner metav1.Object, daemonSet *appsv1.DaemonSet, reqLogger logr.Logger,
	client client.Client, scheme *runtime.Scheme) (*appsv1.DaemonSet, error) {
	var err error

	// Set the owner and controller
	if err = controllerutil.SetControllerReference(owner, daemonSet, scheme); err != nil {
		return nil, err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		toUpdate := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{
			Name:      daemonSet.Name,
			Namespace: daemonSet.Namespace,
			Labels:    map[string]string{},
		}}

		result, err := controllerutil.CreateOrUpdate(context.TODO(), client, toUpdate, func() error {
			toUpdate.Spec = daemonSet.Spec
			for k, v := range daemonSet.Labels {
				toUpdate.Labels[k] = v
			}
			return nil
		})

		if err != nil {
			return err
		}

		if result == controllerutil.OperationResultCreated {
			reqLogger.Info("Created a new DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		} else if result == controllerutil.OperationResultUpdated {
			reqLogger.Info("Updated existing DaemonSet", "DaemonSet.Namespace", daemonSet.Namespace, "DaemonSet.Name", daemonSet.Name)
		}

		return nil
	})

	// Update the status from the server
	if err == nil {
		err = client.Get(context.TODO(), types.NamespacedName{Namespace: daemonSet.Namespace, Name: daemonSet.Name}, daemonSet)
	}

	return daemonSet, errorutil.WithMessagef(err, "error creating or updating DaemonSet %s/%s", daemonSet.Namespace, daemonSet.Name)
}
