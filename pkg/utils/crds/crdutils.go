/*
© 2021 Red Hat, Inc. and others.

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

package crdutils

import (
	"context"
	"fmt"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CRDUpdater interface {
	Create(*apiextensions.CustomResourceDefinition) (*apiextensions.CustomResourceDefinition, error)
	Update(*apiextensions.CustomResourceDefinition) (*apiextensions.CustomResourceDefinition, error)
	Get(string, v1.GetOptions) (*apiextensions.CustomResourceDefinition, error)
	Delete(string, *v1.DeleteOptions) error
}

type controllerClientCreator struct {
	client client.Client
}

func NewFromRestConfig(config *rest.Config) (CRDUpdater, error) {
	apiext, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating the api extensions client: %s", err)
	}
	return NewFromClientSet(apiext), nil
}

func NewFromClientSet(cs clientset.Interface) CRDUpdater {
	return cs.ApiextensionsV1().CustomResourceDefinitions()
}

func NewFromControllerClient(controllerClient client.Client) CRDUpdater {
	return &controllerClientCreator{
		client: controllerClient,
	}
}

func (c *controllerClientCreator) Create(crd *apiextensions.CustomResourceDefinition) (*apiextensions.CustomResourceDefinition, error) {
	err := c.client.Create(context.TODO(), crd)
	return crd, err
}

func (c *controllerClientCreator) Update(crd *apiextensions.CustomResourceDefinition) (*apiextensions.CustomResourceDefinition, error) {
	err := c.client.Update(context.TODO(), crd)
	return crd, err
}

func (c *controllerClientCreator) Get(name string, options v1.GetOptions) (*apiextensions.CustomResourceDefinition, error) {
	crd := &apiextensions.CustomResourceDefinition{}
	err := c.client.Get(context.TODO(), client.ObjectKey{Name: name}, crd)
	if err != nil {
		return nil, err
	}
	return crd, nil
}

func (c *controllerClientCreator) Delete(name string, options *v1.DeleteOptions) error {
	crd, err := c.Get(name, v1.GetOptions{})
	if err != nil {
		return err
	}
	if crd == nil {
		return nil
	}
	return c.client.Delete(context.TODO(), crd)
}
