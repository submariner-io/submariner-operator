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
	Create(context.Context, *apiextensions.CustomResourceDefinition, v1.CreateOptions) (*apiextensions.CustomResourceDefinition, error)
	Update(context.Context, *apiextensions.CustomResourceDefinition, v1.UpdateOptions) (*apiextensions.CustomResourceDefinition, error)
	Get(context.Context, string, v1.GetOptions) (*apiextensions.CustomResourceDefinition, error)
	Delete(context.Context, string, v1.DeleteOptions) error
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

func (c *controllerClientCreator) Create(ctx context.Context, crd *apiextensions.CustomResourceDefinition,
	options v1.CreateOptions) (*apiextensions.CustomResourceDefinition, error) {
	// TODO skitt handle options
	err := c.client.Create(ctx, crd)
	return crd, err
}

func (c *controllerClientCreator) Update(ctx context.Context, crd *apiextensions.CustomResourceDefinition,
	options v1.UpdateOptions) (*apiextensions.CustomResourceDefinition, error) {
	// TODO skitt handle options
	err := c.client.Update(ctx, crd)
	return crd, err
}

func (c *controllerClientCreator) Get(ctx context.Context, name string,
	options v1.GetOptions) (*apiextensions.CustomResourceDefinition, error) {
	crd := &apiextensions.CustomResourceDefinition{}
	// TODO skitt handle options
	err := c.client.Get(ctx, client.ObjectKey{Name: name}, crd)
	if err != nil {
		return nil, err
	}
	return crd, nil
}

func (c *controllerClientCreator) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	crd, err := c.Get(ctx, name, v1.GetOptions{})
	if err != nil {
		return err
	}
	if crd == nil {
		return nil
	}
	// TODO skitt handle options
	return c.client.Delete(ctx, crd)
}
