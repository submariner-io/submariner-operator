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

package datafile

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/stringset"
	"github.com/submariner-io/submariner-operator/internal/component"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/internal/rbac"
	submarinerClientset "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type SubctlData struct {
	BrokerURL        string     `json:"brokerURL"`
	ClientToken      *v1.Secret `omitempty,json:"clientToken"`
	IPSecPSK         *v1.Secret `omitempty,json:"ipsecPSK"`
	ServiceDiscovery bool       `omitempty,json:"serviceDiscovery"`
	Components       []string   `json:",omitempty"`
	CustomDomains    *[]string  `omitempty,json:"customDomains"`
	// Todo (revisit): The following values are moved from the broker-info.subm file to configMap
	// on the Broker. This needs to be revisited to support seamless upgrades.
	// https://github.com/submariner-io/submariner-operator/issues/504
	// GlobalnetCidrRange   string `omitempty,json:"globalnetCidrRange"`
	// GlobalnetClusterSize uint   `omitempty,json:"globalnetClusterSize"`
}

func (data *SubctlData) SetComponents(componentSet stringset.Interface) {
	data.Components = componentSet.Elements()
}

func (data *SubctlData) GetComponents() stringset.Interface {
	return stringset.New(data.Components...)
}

func (data *SubctlData) IsConnectivityEnabled() bool {
	return data.GetComponents().Contains(component.Connectivity)
}

func (data *SubctlData) IsServiceDiscoveryEnabled() bool {
	return data.GetComponents().Contains(component.ServiceDiscovery) || data.ServiceDiscovery
}

func (data *SubctlData) ToString() (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", errors.Wrap(err, "error marshalling data")
	}

	return base64.URLEncoding.EncodeToString(jsonBytes), nil
}

func NewFromString(str string) (*SubctlData, error) {
	data := &SubctlData{}

	bytes, err := base64.URLEncoding.DecodeString(str)
	if err != nil {
		return nil, errors.Wrapf(err, "error decoding %q", str)
	}

	return data, errors.Wrap(json.Unmarshal(bytes, data), "error unmarshalling data")
}

func (data *SubctlData) WriteToFile(filename string) error {
	dataStr, err := data.ToString()
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, []byte(dataStr), 0o600); err != nil {
		return errors.Wrapf(err, "error writing to file %q", filename)
	}

	return nil
}

func NewFromFile(filename string) (*SubctlData, error) {
	dat, err := os.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading file %q", filename)
	}

	return NewFromString(string(dat))
}

func NewFromCluster(restConfig *rest.Config, brokerNamespace, ipsecSubmFile string) (*SubctlData, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "error creating client")
	}

	subCtlData, err := newFromCluster(clientSet, brokerNamespace, ipsecSubmFile)
	if err != nil {
		return nil, err
	}

	subCtlData.BrokerURL = restConfig.Host + restConfig.APIPath

	return subCtlData, err
}

func newFromCluster(clientSet clientset.Interface, brokerNamespace, ipsecSubmFile string) (*SubctlData, error) {
	subctlData := &SubctlData{}
	var err error

	subctlData.ClientToken, err = rbac.GetClientTokenSecret(clientSet, brokerNamespace, constants.SubmarinerBrokerAdminSA)
	if err != nil {
		return nil, errors.Wrap(err, "error getting broker client secret")
	}

	if ipsecSubmFile != "" {
		datafile, err := NewFromFile(ipsecSubmFile)
		if err != nil {
			return nil, errors.Wrapf(err, "error happened trying to import IPsec PSK from subm file: %s", ipsecSubmFile)
		}

		subctlData.IPSecPSK = datafile.IPSecPSK

		return subctlData, err
	}

	subctlData.IPSecPSK, err = newIPSECPSKSecret()

	return subctlData, err
}

func (data *SubctlData) GetBrokerAdministratorConfig() (*rest.Config, error) {
	// We need to try a connection to determine whether the trust chain needs to be provided
	config, err := data.getAndCheckBrokerAdministratorConfig(false)
	if resource.IsUnknownAuthorityError(err) {
		// Certificate error, try with the trust chain
		config, err = data.getAndCheckBrokerAdministratorConfig(true)
	}

	return config, err
}

func (data *SubctlData) getAndCheckBrokerAdministratorConfig(private bool) (*rest.Config, error) {
	config := data.getBrokerAdministratorConfig(private)

	submClientset, err := submarinerClientset.NewForConfig(config)
	if err != nil {
		return config, errors.Wrap(err, "error creating client")
	}

	// This attempts to determine whether we can connect, by trying to access a Submariner object
	// Successful connections result in either the object, or a “not found” error; anything else
	// likely means we couldn’t connect
	_, err = submClientset.SubmarinerV1().Clusters(string(data.ClientToken.Data["namespace"])).List(
		context.TODO(), metav1.ListOptions{})
	if apierrors.IsNotFound(err) {
		err = nil
	}

	return config, err
}

func (data *SubctlData) getBrokerAdministratorConfig(private bool) *rest.Config {
	tlsClientConfig := rest.TLSClientConfig{}
	if private {
		tlsClientConfig.CAData = data.ClientToken.Data["ca.crt"]
	}

	bearerToken := data.ClientToken.Data["token"]
	restConfig := rest.Config{
		Host:            data.BrokerURL,
		TLSClientConfig: tlsClientConfig,
		BearerToken:     string(bearerToken),
	}

	return &restConfig
}
