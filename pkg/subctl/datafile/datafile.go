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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"

	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/broker"
)

type SubctlData struct {
	BrokerURL        string     `json:"brokerURL"`
	ClientToken      *v1.Secret `omitempty,json:"clientToken"`
	IPSecPSK         *v1.Secret `omitempty,json:"ipsecPSK"`
	ServiceDiscovery bool       `omitempty,json:"serviceDiscovery"`
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

	subctlData.ClientToken, err = broker.GetClientTokenSecret(clientSet, brokerNamespace)
	if err != nil {
		return nil, err
	}

	if ipsecSubmFile != "" {
		datafile, err := NewFromFile(ipsecSubmFile)
		if err != nil {
			return nil, fmt.Errorf("Error happened trying to import IPSEC PSK from subm file: %s: %s", ipsecSubmFile,
				err.Error())
		}
		subctlData.IPSecPSK = datafile.IPSecPSK
		return subctlData, err
	} else {
		subctlData.IPSecPSK, err = newIPSECPSKSecret()
		return subctlData, err
	}
}
