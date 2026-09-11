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

package webhook

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"github.com/submariner-io/admiral/pkg/log"
	"github.com/submariner-io/submariner-operator/pkg/names"
	submarinerv1 "github.com/submariner-io/submariner/pkg/apis/submariner.io/v1"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	mcsv1b1 "sigs.k8s.io/mcs-api/pkg/apis/v1beta1"
)

const (
	// SigningRequestLabelKey matches admiral's certificate package constant.
	SigningRequestLabelKey   = "submariner.io/csr-request"
	NotBrokerClientSAMessage = "not a broker-client service account"
)

var webhookLog = logf.Log.WithName("broker-validator")

// BrokerValidator validates secret operations in the broker namespace
// to ensure spoke clusters only access their own certificate secrets.
type BrokerValidator struct {
	decoder admission.Decoder
}

func NewBrokerValidator() *BrokerValidator {
	return &BrokerValidator{
		decoder: admission.NewDecoder(NewScheme()),
	}
}

// SetupWithManager registers the webhook with the manager.
func (v *BrokerValidator) SetupWithManager(mgr ctrl.Manager) {
	mgr.GetWebhookServer().Register("/validate", &admission.Webhook{Handler: v})
}

//nolint:gocritic // Ignore hugeParam - interface method
func (v *BrokerValidator) Handle(_ context.Context, req admission.Request) admission.Response {
	// Extract cluster ID from the ServiceAccount
	clusterID, err := v.extractClusterID(req.UserInfo.Username, req.Namespace)
	if err != nil {
		webhookLog.V(log.DEBUG).Info("Request not from broker-client SA, allowing", "username",
			req.UserInfo.Username, "kind", req.Kind.Kind)

		// Not a broker-client SA, allow (e.g., broker-admin, system:serviceaccount:kube-system:*)
		return admission.Allowed(NotBrokerClientSAMessage)
	}

	if req.Operation != admissionv1.Create && req.Operation != admissionv1.Update && req.Operation != admissionv1.Delete {
		return admission.Allowed("operation not restricted")
	}

	switch req.Kind.Kind {
	case "Secret":
		return v.handleSecret(&req, clusterID)
	case "Cluster":
		return v.handleCluster(&req, clusterID)
	case "Endpoint":
		return v.handleEndpoint(&req, clusterID)
	case "EndpointSlice":
		return v.handleEndpointSlice(&req, clusterID)
	case "ServiceImport":
		return v.handleServiceImport(&req, clusterID)
	default:
		webhookLog.Info("Unexpected resource kind, allowing", "kind", req.Kind.Kind)
		return admission.Allowed("resource kind not validated")
	}
}

// handleSecret validates secret admission requests.
func (v *BrokerValidator) handleSecret(req *admission.Request, clusterID string) admission.Response {
	secret := &corev1.Secret{}

	if err := v.decode(req, secret); err != nil {
		webhookLog.Error(err, "Failed to decode Secret")
		return admission.Errored(http.StatusBadRequest, err)
	}

	// For Update operations, validate that the old owner also matches to prevent ownership takeover
	if req.Operation == admissionv1.Update {
		oldSecret := &corev1.Secret{}
		if err := v.decoder.DecodeRaw(req.OldObject, oldSecret); err != nil {
			webhookLog.Error(err, "Failed to decode old Secret")
			return admission.Errored(http.StatusBadRequest, err)
		}

		oldOwner, oldHasLabel := oldSecret.Labels[SigningRequestLabelKey]
		newOwner, newHasLabel := secret.Labels[SigningRequestLabelKey]

		// Deny updates to unlabeled secrets (shared resources like submariner-ipsec-psk)
		if !oldHasLabel {
			msg := fmt.Sprintf("cluster %s cannot update unlabeled Secret %s", clusterID, secret.Name)
			webhookLog.Info("Denying update of unlabeled Secret", "requesting-cluster", clusterID,
				"secret", secret.Name)

			return admission.Denied(msg)
		}

		// Old label exists - require it to match clusterID
		if oldOwner != clusterID {
			msg := fmt.Sprintf("cluster %s cannot update Secret belonging to cluster %s", clusterID, oldOwner)
			webhookLog.Info("Denying update of other cluster's Secret", "requesting-cluster", clusterID,
				"secret-owner", oldOwner, "secret", secret.Name)

			return admission.Denied(msg)
		}

		// Prevent changing ownership to another cluster
		if newHasLabel && newOwner != clusterID {
			msg := fmt.Sprintf("cluster %s cannot change Secret ownership to cluster %s", clusterID, newOwner)
			webhookLog.Info("Denying Secret ownership change", "requesting-cluster", clusterID,
				"new-owner", newOwner, "secret", secret.Name)

			return admission.Denied(msg)
		}
	}

	// Allow access to secrets that belong to this cluster (certificate secrets)
	// These are labeled with submariner.io/csr-request: <cluster-id>
	if labelValue, ok := secret.Labels[SigningRequestLabelKey]; ok {
		if labelValue == clusterID {
			webhookLog.V(log.DEBUG).Info("Allowing certificate Secret access", "cluster", clusterID,
				"secret", secret.Name, "operation", req.Operation)

			return admission.Allowed(fmt.Sprintf("cluster %s accessing own certificate Secret", clusterID))
		}

		// Attempting to access another cluster's certificate secret
		msg := fmt.Sprintf("cluster %s cannot access certificate Secret belonging to cluster %s",
			clusterID, labelValue)
		webhookLog.Info("Denying cross-cluster secret access", "requesting-cluster", clusterID,
			"secret-owner", labelValue, "secret", secret.Name, "operation", req.Operation)

		return admission.Denied(msg)
	}

	// Deny access to unlabeled secrets (e.g., submariner-ipsec-psk, token secrets)
	// These should only be accessed by broker-admin
	msg := fmt.Sprintf("cluster %s cannot access shared secret %s (no cluster label)", clusterID, secret.Name)
	webhookLog.Info("Denying access to unlabeled secret", "cluster", clusterID, "secret", secret.Name,
		"operation", req.Operation)

	return admission.Denied(msg)
}

// handleCluster validates Cluster admission requests.
func (v *BrokerValidator) handleCluster(req *admission.Request, clusterID string) admission.Response {
	// Validate that the Cluster's name matches the requesting cluster's ID
	if req.Name != clusterID {
		msg := fmt.Sprintf("cluster %s cannot %s Cluster for cluster %s", clusterID, req.Operation, req.Name)
		webhookLog.Info("Denying cross-cluster Cluster access", "requesting-cluster", clusterID,
			"remote-cluster", req.Name, "operation", req.Operation)

		return admission.Denied(msg)
	}

	webhookLog.V(log.DEBUG).Info("Allowing Cluster access", "cluster", clusterID, "operation", req.Operation)

	return admission.Allowed(fmt.Sprintf("cluster %s accessing own Cluster", clusterID))
}

// handleEndpoint validates endpoint admission requests.
func (v *BrokerValidator) handleEndpoint(req *admission.Request, clusterID string) admission.Response {
	endpoint := &submarinerv1.Endpoint{}

	if err := v.decode(req, endpoint); err != nil {
		webhookLog.Error(err, "Failed to decode Endpoint")

		return admission.Errored(http.StatusBadRequest, err)
	}

	// For Update operations, validate that the old ClusterID also matches to prevent ownership takeover
	if req.Operation == admissionv1.Update {
		oldEndpoint := &submarinerv1.Endpoint{}
		if err := v.decoder.DecodeRaw(req.OldObject, oldEndpoint); err != nil {
			webhookLog.Error(err, "Failed to decode old Endpoint")
			return admission.Errored(http.StatusBadRequest, err)
		}

		if oldEndpoint.Spec.ClusterID != clusterID {
			msg := fmt.Sprintf("cluster %s cannot update Endpoint belonging to cluster %s",
				clusterID, oldEndpoint.Spec.ClusterID)
			webhookLog.Info("Denying update of other cluster's Endpoint", "requesting-cluster", clusterID,
				"endpoint-cluster", oldEndpoint.Spec.ClusterID, "endpoint", endpoint.Name)

			return admission.Denied(msg)
		}
	}

	// Validate that the endpoint's ClusterID matches the requesting cluster's ID
	if endpoint.Spec.ClusterID != clusterID {
		msg := fmt.Sprintf("cluster %s cannot %s Endpoint for cluster %s", clusterID, req.Operation, endpoint.Spec.ClusterID)
		webhookLog.Info("Denying cross-cluster Endpoint access", "requesting-cluster", clusterID,
			"endpoint-cluster", endpoint.Spec.ClusterID, "endpoint", endpoint.Name, "operation", req.Operation)

		return admission.Denied(msg)
	}

	webhookLog.V(log.DEBUG).Info("Allowing Endpoint access", "cluster", clusterID, "endpoint", endpoint.Name, "operation", req.Operation)

	return admission.Allowed(fmt.Sprintf("cluster %s accessing own Endpoint", clusterID))
}

// handleEndpointSlice validates EndpointSlice admission requests.
// EndpointSlices are synced by lighthouse to the broker namespace.
func (v *BrokerValidator) handleEndpointSlice(req *admission.Request, clusterID string) admission.Response {
	endpointSlice := &discoveryv1.EndpointSlice{}

	if err := v.decode(req, endpointSlice); err != nil {
		webhookLog.Error(err, "Failed to decode EndpointSlice")

		return admission.Errored(http.StatusBadRequest, err)
	}

	// For Update operations, validate that the old source cluster label also matches to prevent ownership takeover
	if req.Operation == admissionv1.Update {
		oldEndpointSlice := &discoveryv1.EndpointSlice{}
		if err := v.decoder.DecodeRaw(req.OldObject, oldEndpointSlice); err != nil {
			webhookLog.Error(err, "Failed to decode old EndpointSlice")
			return admission.Errored(http.StatusBadRequest, err)
		}

		oldSourceCluster, oldHasLabel := oldEndpointSlice.Labels[mcsv1b1.LabelSourceCluster]

		// Deny updates to unlabeled EndpointSlices
		if !oldHasLabel {
			msg := fmt.Sprintf("cluster %s cannot update unlabeled EndpointSlice %s", clusterID, endpointSlice.Name)
			webhookLog.Info("Denying update of unlabeled EndpointSlice", "requesting-cluster", clusterID,
				"endpointslice", endpointSlice.Name)

			return admission.Denied(msg)
		}

		// Old label exists - require it to match clusterID
		if oldSourceCluster != clusterID {
			msg := fmt.Sprintf("cluster %s cannot update EndpointSlice belonging to cluster %s",
				clusterID, oldSourceCluster)
			webhookLog.Info("Denying update of other cluster's EndpointSlice", "requesting-cluster", clusterID,
				"source-cluster", oldSourceCluster, "endpointslice", endpointSlice.Name)

			return admission.Denied(msg)
		}
	}

	// Validate that the EndpointSlice's source cluster label matches the requesting cluster's ID
	sourceCluster, ok := endpointSlice.Labels[mcsv1b1.LabelSourceCluster]
	if !ok {
		msg := fmt.Sprintf("EndpointSlice %s missing %s label", endpointSlice.Name, mcsv1b1.LabelSourceCluster)
		webhookLog.Info("Denying EndpointSlice without source cluster label", "endpointslice", endpointSlice.Name,
			"operation", req.Operation)

		return admission.Denied(msg)
	}

	if sourceCluster != clusterID {
		msg := fmt.Sprintf("cluster %s cannot %s EndpointSlice for cluster %s",
			clusterID, req.Operation, sourceCluster)
		webhookLog.Info("Denying cross-cluster EndpointSlice access", "requesting-cluster", clusterID,
			"source-cluster", sourceCluster, "endpointslice", endpointSlice.Name, "operation", req.Operation)

		return admission.Denied(msg)
	}

	webhookLog.V(log.DEBUG).Info("Allowing EndpointSlice access", "cluster", clusterID,
		"endpointslice", endpointSlice.Name, "operation", req.Operation)

	return admission.Allowed(fmt.Sprintf("cluster %s accessing own EndpointSlice", clusterID))
}

// handleServiceImport validates ServiceImport admission requests.
// ServiceImports are synced by lighthouse to the broker namespace.
// Aggregated ServiceImports (created by lighthouse aggregation) have LabelServiceName in annotations
// and should be ignored. Local ServiceImports have LabelServiceName in labels.
func (v *BrokerValidator) handleServiceImport(req *admission.Request, clusterID string) admission.Response {
	serviceImport := &mcsv1b1.ServiceImport{}

	if err := v.decode(req, serviceImport); err != nil {
		webhookLog.Error(err, "Failed to decode ServiceImport")

		return admission.Errored(http.StatusBadRequest, err)
	}

	var oldServiceImport *mcsv1b1.ServiceImport

	// For Update operations, validate that the old classification and ownership match to prevent takeover
	if req.Operation == admissionv1.Update {
		oldServiceImport = &mcsv1b1.ServiceImport{}
		if err := v.decoder.DecodeRaw(req.OldObject, oldServiceImport); err != nil {
			webhookLog.Error(err, "Failed to decode old ServiceImport")
			return admission.Errored(http.StatusBadRequest, err)
		}

		// Check if classification changed (local <-> aggregated)
		oldIsAggregated := oldServiceImport.Annotations[mcsv1b1.LabelServiceName] != ""
		newIsAggregated := serviceImport.Annotations[mcsv1b1.LabelServiceName] != ""

		if oldIsAggregated != newIsAggregated {
			msg := fmt.Sprintf("cluster %s cannot change ServiceImport classification", clusterID)
			webhookLog.Info("Denying ServiceImport classification change", "requesting-cluster", clusterID,
				"serviceimport", serviceImport.Name, "old-aggregated", oldIsAggregated, "new-aggregated", newIsAggregated)

			return admission.Denied(msg)
		}

		// For local ServiceImports, validate old ownership
		if !oldIsAggregated {
			oldSourceCluster, oldHasLabel := oldServiceImport.Labels[mcsv1b1.LabelSourceCluster]

			// Deny updates to unlabeled local ServiceImports
			if !oldHasLabel {
				msg := fmt.Sprintf("cluster %s cannot update unlabeled ServiceImport %s", clusterID, serviceImport.Name)
				webhookLog.Info("Denying update of unlabeled ServiceImport", "requesting-cluster", clusterID,
					"serviceimport", serviceImport.Name)

				return admission.Denied(msg)
			}

			// Old label exists - require it to match clusterID
			if oldSourceCluster != clusterID {
				msg := fmt.Sprintf("cluster %s cannot update ServiceImport belonging to cluster %s",
					clusterID, oldSourceCluster)
				webhookLog.Info("Denying update of other cluster's ServiceImport", "requesting-cluster", clusterID,
					"source-cluster", oldSourceCluster, "serviceimport", serviceImport.Name)

				return admission.Denied(msg)
			}
		}
	}

	// Aggregated ServiceImports have LabelServiceName in annotations instead of labels.
	if serviceImport.Annotations[mcsv1b1.LabelServiceName] != "" {
		return v.handleAggregatedServiceImport(req, serviceImport, oldServiceImport, clusterID)
	}

	// Validate that the ServiceImport's source cluster label matches the requesting cluster's ID
	sourceCluster, ok := serviceImport.Labels[mcsv1b1.LabelSourceCluster]
	if !ok {
		msg := fmt.Sprintf("ServiceImport %s missing %s label", serviceImport.Name, mcsv1b1.LabelSourceCluster)
		webhookLog.Info("Denying ServiceImport without source cluster label", "serviceimport", serviceImport.Name,
			"operation", req.Operation)

		return admission.Denied(msg)
	}

	if sourceCluster != clusterID {
		msg := fmt.Sprintf("cluster %s cannot %s ServiceImport for cluster %s", clusterID, req.Operation, sourceCluster)
		webhookLog.Info("Denying cross-cluster ServiceImport access", "requesting-cluster", clusterID,
			"source-cluster", sourceCluster, "serviceimport", serviceImport.Name, "operation", req.Operation)

		return admission.Denied(msg)
	}

	webhookLog.V(log.DEBUG).Info("Allowing ServiceImport access", "cluster", clusterID, "serviceimport", serviceImport.Name,
		"operation", req.Operation)

	return admission.Allowed(fmt.Sprintf("cluster %s accessing own ServiceImport", clusterID))
}

func (v *BrokerValidator) handleAggregatedServiceImport(req *admission.Request, serviceImport, old *mcsv1b1.ServiceImport, clusterID string,
) admission.Response {
	if req.Operation == admissionv1.Update {
		if !reflect.DeepEqual(old.Status.Clusters, serviceImport.Status.Clusters) {
			return v.handleServiceImportClustersUpdated(old.Status.Clusters, serviceImport.Status.Clusters, serviceImport.Name, clusterID)
		}
	}

	ownClusterPresent := slices.ContainsFunc(serviceImport.Status.Clusters, func(c mcsv1b1.ClusterStatus) bool {
		return c.Cluster == clusterID
	})

	if !ownClusterPresent {
		if (req.Operation == admissionv1.Create || req.Operation == admissionv1.Delete) && len(serviceImport.Status.Clusters) == 0 {
			// Because Status is a subresource, the resource must first be created w/o Status.
			return admission.Allowed("")
		}

		webhookLog.Info("Denying aggregated ServiceImport with only other clusters present", "serviceimport", serviceImport.Name,
			"operation", req.Operation)

		return admission.Denied(fmt.Sprintf("Aggregated ServiceImport %s missing own cluster", serviceImport.Name))
	}

	otherClusterPresent := len(serviceImport.Status.Clusters) > 1
	if otherClusterPresent && (req.Operation == admissionv1.Create || req.Operation == admissionv1.Delete) {
		webhookLog.Info("Denying aggregated ServiceImport with other clusters present", "serviceimport", serviceImport.Name,
			"operation", req.Operation)

		return admission.Denied(fmt.Sprintf("Aggregated ServiceImport %s contains other clusters", serviceImport.Name))
	}

	return admission.Allowed("")
}

func (v *BrokerValidator) handleServiceImportClustersUpdated(old, updated []mcsv1b1.ClusterStatus, name, clusterID string,
) admission.Response {
	otherClusterNames := func(clusters []mcsv1b1.ClusterStatus) []string {
		names := make([]string, 0, len(clusters))
		for _, c := range clusters {
			if c.Cluster != clusterID {
				names = append(names, c.Cluster)
			}
		}

		slices.Sort(names)

		return names
	}

	if !slices.Equal(otherClusterNames(updated), otherClusterNames(old)) {
		webhookLog.Info("Denying aggregated ServiceImport - modifying other clusters", "serviceimport", name,
			"updated", fmt.Sprintf("%v", updated), "old", fmt.Sprintf("%v", old))

		return admission.Denied(fmt.Sprintf("Aggregated ServiceImport %s attempting to modify other clusters", name))
	}

	return admission.Allowed("")
}

// extractClusterID extracts the cluster ID from a broker-client ServiceAccount username.
// Expected format: system:serviceaccount:<namespace>:cluster-<id>
// This matches the format created by subctl via names.ForClusterSA(clusterID).
// Returns cluster ID and nil on success, empty string and error if not a broker-client SA.
func (v *BrokerValidator) extractClusterID(username, brokerNamespace string) (string, error) {
	// Format: system:serviceaccount:<namespace>:<sa-name>
	parts := strings.Split(username, ":")
	if len(parts) != 4 || parts[0] != "system" || parts[1] != "serviceaccount" {
		return "", errors.New("not a service account")
	}

	// Validate that the ServiceAccount is from the broker namespace
	// to prevent a SA from another namespace claiming to be a cluster
	if parts[2] != brokerNamespace {
		return "", errors.New("service account is outside the broker namespace")
	}

	saName := parts[3]

	// Check if it matches cluster-<id> pattern (subctl creates SAs via names.ForClusterSA)
	if !strings.HasPrefix(saName, names.ClusterSAPrefix) {
		return "", errors.New("not a broker-client service account")
	}

	// Extract cluster ID from cluster-<id>
	clusterID := strings.TrimPrefix(saName, names.ClusterSAPrefix)

	if clusterID == "" {
		return "", errors.New("empty cluster ID")
	}

	return clusterID, nil
}

//nolint:wrapcheck //  No need to wrap
func (v *BrokerValidator) decode(req *admission.Request, into runtime.Object) error {
	if req.Operation == admissionv1.Delete {
		return v.decoder.DecodeRaw(req.OldObject, into)
	}

	return v.decoder.Decode(*req, into)
}

func NewScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(discoveryv1.AddToScheme(scheme))
	utilruntime.Must(mcsv1b1.Install(scheme))
	utilruntime.Must(submarinerv1.AddToScheme(scheme))

	return scheme
}
