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

package submariner

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/ciliumcm"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// reconcileCiliumClusterMesh ensures TLS for the route-agent publisher and merges the
// Submariner peer into Cilium's cilium-clustermesh Secret when NetworkPlugin is cilium.
// Returns CertCheckRequeue so expiry is re-checked even without other events.
func (r *Reconciler) reconcileCiliumClusterMesh(ctx context.Context, instance *v1alpha1.Submariner,
	reqLogger logr.Logger,
) (time.Duration, error) {
	if instance.Status.NetworkPlugin != "cilium" {
		// Drop any leftover Submariner peer so Cilium stops dialing our publisher.
		return 0, r.removeCiliumClusterMeshPeer(ctx, reqLogger)
	}

	tlsSecret, err := r.ensureCiliumCMTLSSecret(ctx, instance, reqLogger)
	if err != nil {
		return 0, err
	}

	ok, msg := r.checkCiliumClusterID(ctx, reqLogger)
	if !ok {
		reqLogger.Info("Skipping cilium-clustermesh peer merge", "reason", msg)

		return ciliumcm.CertCheckRequeue, nil
	}

	if err := r.mergeCiliumClusterMeshPeer(ctx, tlsSecret, reqLogger); err != nil {
		return 0, err
	}

	reqLogger.Info("Ensured Cilium CM publisher TLS and peer",
		"tlsSecret", ciliumcm.TLSSecretName,
		"remote", ciliumcm.DefaultRemoteName,
		"clusterMeshSecret", ciliumcm.ClusterMeshSecretName)

	return ciliumcm.CertCheckRequeue, nil
}

func (r *Reconciler) ensureCiliumCMTLSSecret(ctx context.Context, instance *v1alpha1.Submariner,
	reqLogger logr.Logger,
) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	key := types.NamespacedName{Namespace: instance.Namespace, Name: ciliumcm.TLSSecretName}

	err := r.config.ScopedClient.Get(ctx, key, secret)
	if err == nil {
		action, reason := ciliumcm.AssessSecretData(secret.Data, time.Now(), ciliumcm.DefaultRenewBefore)
		switch action {
		case ciliumcm.RenewNone:
			return secret, nil
		case ciliumcm.RenewLeaves:
			reqLogger.Info("Renewing Cilium CM leaf certificates (keeping CA)", "secret", key.Name, "reason", reason)

			bundle, genErr := ciliumcm.ReissueLeaves(secret.Data[ciliumcm.CACertKey], secret.Data[ciliumcm.CAKeyKey], 0)
			if genErr != nil {
				return nil, errors.Wrap(genErr, "renew Cilium CM leaf certificates")
			}

			return r.updateCiliumCMTLSSecret(ctx, secret, bundle, reqLogger)
		case ciliumcm.RenewFull:
			reqLogger.Info("Regenerating Cilium CM TLS bundle", "secret", key.Name, "reason", reason)
			// fall through to full generate + update below with notFound=false
		}
	} else if !apierrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "get Cilium CM TLS Secret")
	}

	notFound := apierrors.IsNotFound(err)

	bundle, genErr := ciliumcm.GenerateBundle(0)
	if genErr != nil {
		return nil, errors.Wrap(genErr, "generate Cilium CM TLS bundle")
	}

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ciliumcm.TLSSecretName,
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/component": "cilium-cm-publisher",
				"app.kubernetes.io/part-of":   "submariner",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: bundle.SecretData(),
	}

	if err := controllerutil.SetControllerReference(instance, newSecret, r.config.Scheme); err != nil {
		return nil, errors.Wrap(err, "set owner reference on Cilium CM TLS Secret")
	}

	if notFound {
		if err := r.config.ScopedClient.Create(ctx, newSecret); err != nil {
			return nil, errors.Wrap(err, "create Cilium CM TLS Secret")
		}

		reqLogger.Info("Created Cilium CM TLS Secret", "secret", key.Name)

		return newSecret, nil
	}

	return r.updateCiliumCMTLSSecret(ctx, secret, bundle, reqLogger)
}

func (r *Reconciler) updateCiliumCMTLSSecret(ctx context.Context, secret *corev1.Secret, bundle *ciliumcm.Bundle,
	reqLogger logr.Logger,
) (*corev1.Secret, error) {
	secret.Data = bundle.SecretData()
	secret.Type = corev1.SecretTypeOpaque

	if err := r.config.ScopedClient.Update(ctx, secret); err != nil {
		return nil, errors.Wrap(err, "update Cilium CM TLS Secret")
	}

	reqLogger.Info("Updated Cilium CM TLS Secret", "secret", secret.Name)

	return secret, nil
}

func (r *Reconciler) checkCiliumClusterID(ctx context.Context, reqLogger logr.Logger) (bool, string) {
	cm := &corev1.ConfigMap{}

	err := r.config.GeneralClient.Get(ctx, types.NamespacedName{
		Namespace: metav1.NamespaceSystem,
		Name:      ciliumcm.CiliumConfigMapName,
	}, cm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, "cilium-config ConfigMap not found in kube-system; cannot validate cluster-id"
		}

		reqLogger.Error(err, "Failed to get cilium-config")

		return false, fmt.Sprintf("failed to read cilium-config: %v", err)
	}

	clusterID := cm.Data["cluster-id"]
	clusterName := cm.Data["cluster-name"]

	if clusterID == "" || clusterID == "0" {
		return false, "cilium-config cluster-id is 0 or unset; set to 1..255 for ClusterMesh-shaped publisher"
	}

	if clusterName == "" || clusterName == "default" {
		return false, "cilium-config cluster-name is 'default' or unset; set a non-default name"
	}

	return true, ""
}

func (r *Reconciler) mergeCiliumClusterMeshPeer(ctx context.Context, tlsSecret *corev1.Secret, reqLogger logr.Logger) error {
	remoteName := ciliumcm.DefaultRemoteName
	peerKeys := map[string][]byte{
		remoteName:                         ciliumcm.PeerConfigYAML(remoteName, ciliumcm.DefaultListenURL),
		remoteName + ".etcd-client-ca.crt": tlsSecret.Data[ciliumcm.CACertKey],
		remoteName + ".etcd-client.crt":    tlsSecret.Data[ciliumcm.ClientCertKey],
		remoteName + ".etcd-client.key":    tlsSecret.Data[ciliumcm.ClientKeyKey],
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: ciliumcm.ClusterMeshSecretName}

		err := r.config.GeneralClient.Get(ctx, key, secret)
		if apierrors.IsNotFound(err) {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ciliumcm.ClusterMeshSecretName,
					Namespace: metav1.NamespaceSystem,
				},
				Type: corev1.SecretTypeOpaque,
				Data: peerKeys,
			}

			if createErr := r.config.GeneralClient.Create(ctx, secret); createErr != nil {
				return errors.Wrap(createErr, "create cilium-clustermesh Secret")
			}

			reqLogger.Info("Created cilium-clustermesh Secret with Submariner peer", "remote", remoteName)

			return nil
		}

		if err != nil {
			return errors.Wrap(err, "get cilium-clustermesh Secret")
		}

		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}

		changed := false

		for k, v := range peerKeys {
			if !bytes.Equal(secret.Data[k], v) {
				secret.Data[k] = v
				changed = true
			}
		}

		if !changed {
			return nil
		}

		if err := r.config.GeneralClient.Update(ctx, secret); err != nil {
			return errors.Wrap(err, "update cilium-clustermesh Secret")
		}

		reqLogger.Info("Merged Submariner peer into cilium-clustermesh", "remote", remoteName)

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "merge Submariner peer into cilium-clustermesh Secret")
	}

	return nil
}

// removeCiliumClusterMeshPeer removes only Submariner-owned keys from cilium-clustermesh
// (see ciliumcm.PeerSecretKeys). Other ClusterMesh peers are never touched.
//
// Flow: Update first so peer keys disappear even without delete RBAC; if the Secret
// is then empty, Delete it (Cilium mounts the Secret with optional: true).
func (r *Reconciler) removeCiliumClusterMeshPeer(ctx context.Context, reqLogger logr.Logger) error {
	remoteName := ciliumcm.DefaultRemoteName
	ownedKeys := ciliumcm.PeerSecretKeys(remoteName)

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		secret := &corev1.Secret{}
		key := types.NamespacedName{Namespace: metav1.NamespaceSystem, Name: ciliumcm.ClusterMeshSecretName}

		err := r.config.GeneralClient.Get(ctx, key, secret)
		if apierrors.IsNotFound(err) {
			return nil
		}

		if err != nil {
			return errors.Wrap(err, "get cilium-clustermesh Secret")
		}

		if secret.Data == nil {
			return nil
		}

		changed := false

		for _, k := range ownedKeys {
			if _, ok := secret.Data[k]; ok {
				delete(secret.Data, k)

				changed = true
			}
		}

		if !changed {
			return nil
		}

		// Persist key removal before any Delete. Never wipe unrelated peer keys.
		empty := len(secret.Data) == 0

		if err := r.config.GeneralClient.Update(ctx, secret); err != nil {
			return errors.Wrap(err, "update cilium-clustermesh Secret to remove Submariner peer")
		}

		if !empty {
			reqLogger.Info("Removed Submariner peer from cilium-clustermesh", "remote", remoteName)

			return nil
		}

		// Only our keys were present — drop the now-empty Secret (optional for Cilium).
		if err := r.config.GeneralClient.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
			// Keys are already cleared; an empty leftover Secret is harmless.
			reqLogger.Info("Cleared Submariner peer; leaving empty cilium-clustermesh Secret",
				"remote", remoteName, "reason", err.Error())

			return nil
		}

		reqLogger.Info("Removed Submariner peer and deleted empty cilium-clustermesh Secret",
			"remote", remoteName)

		return nil
	})
	if err != nil {
		return errors.Wrap(err, "remove Submariner peer from cilium-clustermesh Secret")
	}

	return nil
}
