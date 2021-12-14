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

package broker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/resource"
	submarinerClientset "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

type Info struct {
	BrokerURL        string         `json:"brokerURL"`
	ClientToken      *corev1.Secret `omitempty,json:"clientToken"`
	IPSecPSK         *corev1.Secret `omitempty,json:"ipsecPSK"`
	ServiceDiscovery bool           `omitempty,json:"serviceDiscovery"`
	Components       []string       `json:",omitempty"`
	CustomDomains    *[]string      `omitempty,json:"customDomains"`
}

func (d *Info) writeToFile(filename string) error {
	dataStr, err := d.encode()
	if err != nil {
		return err
	}

	if err := os.WriteFile(filename, []byte(dataStr), 0o600); err != nil {
		return errors.Wrapf(err, "error writing to file %q", filename)
	}

	return nil
}

func (d *Info) encode() (string, error) {
	jsonBytes, err := json.Marshal(d)
	if err != nil {
		return "", errors.Wrap(err, "error marshalling data")
	}

	return base64.URLEncoding.EncodeToString(jsonBytes), nil
}

func (d *Info) GetBrokerAdministratorConfig() (*rest.Config, error) {
	// We need to try a connection to determine whether the trust chain needs to be provided
	config, err := d.getAndCheckBrokerAdministratorConfig(false)
	if resource.IsUnknownAuthorityError(err) {
		// Certificate error, try with the trust chain
		config, err = d.getAndCheckBrokerAdministratorConfig(true)
	}

	return config, err
}

func (d *Info) getAndCheckBrokerAdministratorConfig(private bool) (*rest.Config, error) {
	config := d.getBrokerAdministratorConfig(private)

	submClientset, err := submarinerClientset.NewForConfig(config)
	if err != nil {
		return config, errors.Wrap(err, "error creating client")
	}

	// This attempts to determine whether we can connect, by trying to access a Submariner object
	// Successful connections result in either the object, or a “not found” error; anything else
	// likely means we couldn’t connect
	_, err = submClientset.SubmarinerV1().Clusters(string(d.ClientToken.Data["namespace"])).List(
		context.TODO(), metav1.ListOptions{})
	if apierrors.IsNotFound(err) {
		err = nil
	}

	return config, err
}

func (d *Info) getBrokerAdministratorConfig(private bool) *rest.Config {
	tlsClientConfig := rest.TLSClientConfig{}
	if private {
		tlsClientConfig.CAData = d.ClientToken.Data["ca.crt"]
	}

	bearerToken := d.ClientToken.Data["token"]
	restConfig := rest.Config{
		Host:            d.BrokerURL,
		TLSClientConfig: tlsClientConfig,
		BearerToken:     string(bearerToken),
	}

	return &restConfig
}
