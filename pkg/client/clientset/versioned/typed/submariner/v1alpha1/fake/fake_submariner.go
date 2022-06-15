/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeSubmariners implements SubmarinerInterface
type FakeSubmariners struct {
	Fake *FakeSubmarinerV1alpha1
	ns   string
}

var submarinersResource = schema.GroupVersionResource{Group: "submariner.io", Version: "v1alpha1", Resource: "submariners"}

var submarinersKind = schema.GroupVersionKind{Group: "submariner.io", Version: "v1alpha1", Kind: "Submariner"}

// Get takes name of the submariner, and returns the corresponding submariner object, and an error if there is any.
func (c *FakeSubmariners) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.Submariner, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(submarinersResource, c.ns, name), &v1alpha1.Submariner{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Submariner), err
}

// List takes label and field selectors, and returns the list of Submariners that match those selectors.
func (c *FakeSubmariners) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.SubmarinerList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(submarinersResource, submarinersKind, c.ns, opts), &v1alpha1.SubmarinerList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.SubmarinerList{ListMeta: obj.(*v1alpha1.SubmarinerList).ListMeta}
	for _, item := range obj.(*v1alpha1.SubmarinerList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested submariners.
func (c *FakeSubmariners) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(submarinersResource, c.ns, opts))

}

// Create takes the representation of a submariner and creates it.  Returns the server's representation of the submariner, and an error, if there is any.
func (c *FakeSubmariners) Create(ctx context.Context, submariner *v1alpha1.Submariner, opts v1.CreateOptions) (result *v1alpha1.Submariner, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(submarinersResource, c.ns, submariner), &v1alpha1.Submariner{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Submariner), err
}

// Update takes the representation of a submariner and updates it. Returns the server's representation of the submariner, and an error, if there is any.
func (c *FakeSubmariners) Update(ctx context.Context, submariner *v1alpha1.Submariner, opts v1.UpdateOptions) (result *v1alpha1.Submariner, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(submarinersResource, c.ns, submariner), &v1alpha1.Submariner{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Submariner), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeSubmariners) UpdateStatus(ctx context.Context, submariner *v1alpha1.Submariner, opts v1.UpdateOptions) (*v1alpha1.Submariner, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(submarinersResource, "status", c.ns, submariner), &v1alpha1.Submariner{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Submariner), err
}

// Delete takes name of the submariner and deletes it. Returns an error if one occurs.
func (c *FakeSubmariners) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(submarinersResource, c.ns, name), &v1alpha1.Submariner{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeSubmariners) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(submarinersResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.SubmarinerList{})
	return err
}

// Patch applies the patch and returns the patched submariner.
func (c *FakeSubmariners) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Submariner, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(submarinersResource, c.ns, name, pt, data, subresources...), &v1alpha1.Submariner{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Submariner), err
}
