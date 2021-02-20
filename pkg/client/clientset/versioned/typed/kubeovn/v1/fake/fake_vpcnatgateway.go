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

	kubeovnv1 "github.com/alauda/kube-ovn/pkg/apis/kubeovn/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeVpcNatGateways implements VpcNatGatewayInterface
type FakeVpcNatGateways struct {
	Fake *FakeKubeovnV1
}

var vpcnatgatewaysResource = schema.GroupVersionResource{Group: "kubeovn.io", Version: "v1", Resource: "vpc-nat-gateways"}

var vpcnatgatewaysKind = schema.GroupVersionKind{Group: "kubeovn.io", Version: "v1", Kind: "VpcNatGateway"}

// Get takes name of the vpcNatGateway, and returns the corresponding vpcNatGateway object, and an error if there is any.
func (c *FakeVpcNatGateways) Get(ctx context.Context, name string, options v1.GetOptions) (result *kubeovnv1.VpcNatGateway, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(vpcnatgatewaysResource, name), &kubeovnv1.VpcNatGateway{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubeovnv1.VpcNatGateway), err
}

// List takes label and field selectors, and returns the list of VpcNatGateways that match those selectors.
func (c *FakeVpcNatGateways) List(ctx context.Context, opts v1.ListOptions) (result *kubeovnv1.VpcNatGatewayList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(vpcnatgatewaysResource, vpcnatgatewaysKind, opts), &kubeovnv1.VpcNatGatewayList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &kubeovnv1.VpcNatGatewayList{ListMeta: obj.(*kubeovnv1.VpcNatGatewayList).ListMeta}
	for _, item := range obj.(*kubeovnv1.VpcNatGatewayList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested vpcNatGateways.
func (c *FakeVpcNatGateways) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(vpcnatgatewaysResource, opts))
}

// Create takes the representation of a vpcNatGateway and creates it.  Returns the server's representation of the vpcNatGateway, and an error, if there is any.
func (c *FakeVpcNatGateways) Create(ctx context.Context, vpcNatGateway *kubeovnv1.VpcNatGateway, opts v1.CreateOptions) (result *kubeovnv1.VpcNatGateway, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(vpcnatgatewaysResource, vpcNatGateway), &kubeovnv1.VpcNatGateway{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubeovnv1.VpcNatGateway), err
}

// Update takes the representation of a vpcNatGateway and updates it. Returns the server's representation of the vpcNatGateway, and an error, if there is any.
func (c *FakeVpcNatGateways) Update(ctx context.Context, vpcNatGateway *kubeovnv1.VpcNatGateway, opts v1.UpdateOptions) (result *kubeovnv1.VpcNatGateway, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(vpcnatgatewaysResource, vpcNatGateway), &kubeovnv1.VpcNatGateway{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubeovnv1.VpcNatGateway), err
}

// Delete takes name of the vpcNatGateway and deletes it. Returns an error if one occurs.
func (c *FakeVpcNatGateways) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteAction(vpcnatgatewaysResource, name), &kubeovnv1.VpcNatGateway{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeVpcNatGateways) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(vpcnatgatewaysResource, listOpts)

	_, err := c.Fake.Invokes(action, &kubeovnv1.VpcNatGatewayList{})
	return err
}

// Patch applies the patch and returns the patched vpcNatGateway.
func (c *FakeVpcNatGateways) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *kubeovnv1.VpcNatGateway, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(vpcnatgatewaysResource, name, pt, data, subresources...), &kubeovnv1.VpcNatGateway{})
	if obj == nil {
		return nil, err
	}
	return obj.(*kubeovnv1.VpcNatGateway), err
}
