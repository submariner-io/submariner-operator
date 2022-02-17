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

package test

import (
	"context"
	"errors"
	"reflect"

	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type FailingClient struct {
	controllerClient.Client
	OnCreate reflect.Type
	OnGet    reflect.Type
	OnUpdate reflect.Type
}

func (c *FailingClient) Create(ctx context.Context, obj controllerClient.Object, opts ...controllerClient.CreateOption) error {
	if c.OnCreate == reflect.TypeOf(obj) {
		return errors.New("mock Create error")
	}

	return c.Client.Create(ctx, obj, opts...)
}

func (c *FailingClient) Get(ctx context.Context, key controllerClient.ObjectKey, obj controllerClient.Object) error {
	if c.OnGet == reflect.TypeOf(obj) {
		return errors.New("mock Get error")
	}

	return c.Client.Get(ctx, key, obj)
}

func (c *FailingClient) Update(ctx context.Context, obj controllerClient.Object, opts ...controllerClient.UpdateOption) error {
	if c.OnUpdate == reflect.TypeOf(obj) {
		return errors.New("mock Update error")
	}

	return c.Client.Update(ctx, obj, opts...)
}
