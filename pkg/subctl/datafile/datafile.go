/*
© 2019 Red Hat, Inc. and others.

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
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/url"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/broker"
	submarinerClientset "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
)

type SubctlData struct {
	BrokerURL        string     `json:"brokerURL"`
	ClientToken      *v1.Secret `omitempty,json:"clientToken"`
	IPSecPSK         *v1.Secret `omitempty,json:"ipsecPSK"`
	ServiceDiscovery bool       `omitempty,json:"serviceDiscovery"`
	// Todo (revisit): The following values are moved from the broker-info.subm file to configMap
	// on the Broker. This needs to be revisited to support seamless upgrades.
	// https://github.com/submariner-io/submariner-operator/issues/504
	// GlobalnetCidrRange   string `omitempty,json:"globalnetCidrRange"`
	// GlobalnetClusterSize uint   `omitempty,json:"globalnetClusterSize"`
}

func (data *SubctlData) ToString() (string, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(jsonBytes), nil
}

func NewFromString(str string) (*SubctlData, error) {
	data := &SubctlData{}
	bytes, err := base64.URLEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return data, json.Unmarshal(bytes, data)
}

func (data *SubctlData) WriteToFile(filename string) error {

	dataStr, err := data.ToString()
	if err != nil {
		return err
	}

	if err = ioutil.WriteFile(filename, []byte(dataStr), 0644); err != nil {
		return err
	}

	return nil
}

func NewFromFile(filename string) (*SubctlData, error) {
	dat, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return NewFromString(string(dat))
}

func NewFromCluster(restConfig *rest.Config, brokerNamespace, ipsecSubmFile string) (*SubctlData, error) {

	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return nil, err
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

	subctlData.ClientToken, err = broker.GetClientTokenSecret(clientSet, brokerNamespace, broker.SubmarinerBrokerAdminSA)
	if err != nil {
		return nil, err
	}

	if ipsecSubmFile != "" {
		datafile, err := NewFromFile(ipsecSubmFile)
		if err != nil {
			return nil, fmt.Errorf("Error happened trying to import IPsec PSK from subm file: %s: %s", ipsecSubmFile,
				err.Error())
		}
		subctlData.IPSecPSK = datafile.IPSecPSK
		return subctlData, err
	} else {
		subctlData.IPSecPSK, err = newIPSECPSKSecret()
		return subctlData, err
	}
}

func (data *SubctlData) GetBrokerAdministratorConfig() (*rest.Config, error) {
	// We need to try a connection to determine whether the trust chain needs to be provided
	config, err := data.getAndCheckBrokerAdministratorConfig(false)
	if err != nil {
		if urlError, ok := err.(*url.Error); ok {
			if _, ok := urlError.Unwrap().(x509.UnknownAuthorityError); ok {
				// Certificate error, try with the trust chain
				config, err = data.getAndCheckBrokerAdministratorConfig(true)
			}
		}
	}
	return config, err
}

func (data *SubctlData) getAndCheckBrokerAdministratorConfig(private bool) (*rest.Config, error) {
	config := data.getBrokerAdministratorConfig(private)
	clientset, err := submarinerClientset.NewForConfig(config)
	if err != nil {
		return config, err
	}
	// The point of the broker client is to access Submariner data, we know we should able to list clusters
	_, err = clientset.SubmarinerV1().Clusters(string(data.ClientToken.Data["namespace"])).List(metav1.ListOptions{})
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
