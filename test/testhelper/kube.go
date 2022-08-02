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

package testhelper

import (
	"context"
	"fmt"
	"github.com/labring/endpoints-operator/api/network/v1beta1"
	"github.com/labring/endpoints-operator/client"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sync"
	"testing"
)

var once sync.Once
var cep *client.Cep

func cepClient() {
	once.Do(func() {
		op := client.NewKubernetesOptions("", "")
		c := client.NewKubernetesClient(op)
		cep = client.NewCep(c.KubernetesDynamic())
	})
}

func CreateTestCRs(num int, prefix string, ep *v1beta1.ClusterEndpoint, t *testing.T) {
	ctx := context.Background()
	cepClient()
	ep.Labels = map[string]string{"test": prefix}
	for i := 0; i < num; i++ {
		ep.Name = fmt.Sprintf("%s-%d", prefix, i)
		err := cep.CreateCR(ctx, ep.DeepCopy())
		if err != nil {
			t.Errorf("%s create failed: %v", ep.Name, err)
			continue
		}
		t.Logf("%s create success.", ep.Name)
	}
}

func DeleteTestCRs(prefix, namespace string, t *testing.T) {
	m := map[string]string{"test": prefix}
	cepClient()
	ctx := context.Background()
	err := cep.DeleteCRs(ctx, namespace, v1.ListOptions{
		LabelSelector: labels.FormatLabels(m),
	})
	if err != nil {
		t.Errorf("delete failed: %v", err)
		return
	}
	t.Log("delete success.")
}
