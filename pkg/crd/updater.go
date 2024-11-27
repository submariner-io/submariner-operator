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

//nolint:wrapcheck // These functions are basically wrappers for the k8s APIs.
package crd

import (
	"context"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/submariner-operator/pkg/embeddedyamls"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type baseUpdater interface {
	Create(ctx context.Context, crd *apiextensions.CustomResourceDefinition, options metav1.CreateOptions,
	) (*apiextensions.CustomResourceDefinition, error)
	Update(ctx context.Context, crd *apiextensions.CustomResourceDefinition, options metav1.UpdateOptions,
	) (*apiextensions.CustomResourceDefinition, error)
	Get(ctx context.Context, name string, options metav1.GetOptions) (*apiextensions.CustomResourceDefinition, error)
	Delete(ctx context.Context, name string, options metav1.DeleteOptions) error
}

type Updater interface {
	baseUpdater
	CreateOrUpdateFromEmbedded(ctx context.Context, name string) (bool, error)
}

type updater struct {
	baseUpdater
}

type controllerClientCreator struct {
	client client.Client
}

func UpdaterFromRestConfig(config *rest.Config) (Updater, error) {
	apiext, err := clientset.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "error creating the api extensions client")
	}

	return UpdaterFromClientSet(apiext), nil
}

func UpdaterFromClientSet(cs clientset.Interface) Updater {
	return &updater{baseUpdater: cs.ApiextensionsV1().CustomResourceDefinitions()}
}

func UpdaterFromControllerClient(controllerClient client.Client) Updater {
	return &updater{baseUpdater: &controllerClientCreator{
		client: controllerClient,
	}}
}

func (u *updater) CreateOrUpdateFromEmbedded(ctx context.Context, crdYaml string) (bool, error) {
	crd := &apiextensions.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(crdYaml, crd); err != nil {
		return false, errors.Wrap(err, "error extracting embedded CRD")
	}

	result, err := util.CreateOrUpdate[*apiextensions.CustomResourceDefinition](
		ctx, &resource.InterfaceFuncs[*apiextensions.CustomResourceDefinition]{
			GetFunc:    u.Get,
			CreateFunc: u.Create,
			UpdateFunc: u.Update,
		}, crd, util.Replace(crd))

	return result == util.OperationResultCreated, err
}

func (c *controllerClientCreator) Create(ctx context.Context, crd *apiextensions.CustomResourceDefinition,
	opts metav1.CreateOptions, //nolint:gocritic // hugeParam - match K8s API
) (*apiextensions.CustomResourceDefinition, error) {
	err := c.client.Create(ctx, crd, &client.CreateOptions{
		DryRun:       opts.DryRun,
		FieldManager: opts.FieldManager,
		Raw:          &opts,
	})

	return crd, err
}

func (c *controllerClientCreator) Update(ctx context.Context, crd *apiextensions.CustomResourceDefinition,
	opts metav1.UpdateOptions, //nolint:gocritic // hugeParam - match K8s API
) (*apiextensions.CustomResourceDefinition, error) {
	err := c.client.Update(ctx, crd, &client.UpdateOptions{
		DryRun:       opts.DryRun,
		FieldManager: opts.FieldManager,
		Raw:          &opts,
	})

	return crd, err
}

func (c *controllerClientCreator) Get(ctx context.Context, name string,
	opts metav1.GetOptions,
) (*apiextensions.CustomResourceDefinition, error) {
	crd := &apiextensions.CustomResourceDefinition{}

	err := c.client.Get(ctx, client.ObjectKey{Name: name}, crd, &client.GetOptions{Raw: &opts})
	if err != nil {
		return nil, err
	}

	return crd, nil
}

func (c *controllerClientCreator) Delete(ctx context.Context, name string,
	opts metav1.DeleteOptions, //nolint:gocritic // Match K8s API
) error {
	crd, err := c.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if crd == nil {
		return nil
	}

	return c.client.Delete(ctx, crd, &client.DeleteOptions{
		DryRun:             opts.DryRun,
		GracePeriodSeconds: opts.GracePeriodSeconds,
		Preconditions:      opts.Preconditions,
		PropagationPolicy:  opts.PropagationPolicy,
		Raw:                &opts,
	})
}
