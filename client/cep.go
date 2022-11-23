// Copyright © 2022 The sealos Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"

	"github.com/labring/endpoints-operator/api/network/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type Cep struct {
	gvr    schema.GroupVersionResource
	client dynamic.Interface
}

func NewCep(client dynamic.Interface) *Cep {
	c := &Cep{}
	c.gvr = schema.GroupVersionResource{Group: v1beta1.GroupName, Version: v1beta1.GroupVersion.Version, Resource: "clusterendpoints"}
	//NewKubernetesClient(NewKubernetesOptions("", ""))
	c.client = client
	return c
}

func (c *Cep) CreateCR(ctx context.Context, endpoint *v1beta1.ClusterEndpoint) error {
	endpoint.APIVersion = v1beta1.GroupVersion.String()
	endpoint.Kind = "ClusterEndpoint"
	_, err := c.client.Resource(c.gvr).Namespace(endpoint.Namespace).Create(ctx, runtimeConvertUnstructured(endpoint), v1.CreateOptions{})
	return err
}

func (c *Cep) DeleteCR(ctx context.Context, namespace, name string) error {
	return c.client.Resource(c.gvr).Namespace(namespace).Delete(ctx, name, v1.DeleteOptions{})
}

func (c *Cep) DeleteCRs(ctx context.Context, namespace string, options v1.ListOptions) error {
	return c.client.Resource(c.gvr).Namespace(namespace).DeleteCollection(ctx, v1.DeleteOptions{}, options)
}
