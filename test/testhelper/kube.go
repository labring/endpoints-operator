/*
Copyright 2022 cuisongliu@qq.com.

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

package testhelper

import (
	"fmt"
	"github.com/sealyun/endpoints-operator/api/network/v1beta1"
	"github.com/sealyun/endpoints-operator/library/convert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
)

var gvr = schema.GroupVersionResource{Group: v1beta1.GroupName, Version: v1beta1.GroupVersion.Version, Resource: "clusterendpoints"}

func client() dynamic.Interface {
	return dynamic.NewForConfigOrDie(ctrl.GetConfigOrDie())
}

func runtimeConvertUnstructured(from runtime.Object) *unstructured.Unstructured {
	to, ok := from.(*unstructured.Unstructured)
	if ok {
		return to
	}
	if to, err := convert.ResourceToUnstructured(from); err == nil {
		return to
	}
	return nil
}

func CreateCR(endpoint *v1beta1.ClusterEndpoint) error {
	endpoint.APIVersion = v1beta1.GroupVersion.String()
	endpoint.Kind = "ClusterEndpoint"
	_, err := client().Resource(gvr).Namespace(endpoint.Namespace).Create(runtimeConvertUnstructured(endpoint), v1.CreateOptions{})
	return err
}

func DeleteCR(namespace, name string) error {
	return client().Resource(gvr).Namespace(namespace).Delete(name, &v1.DeleteOptions{})
}

func DeleteCRs(namespace string, options v1.ListOptions) error {
	return client().Resource(gvr).Namespace(namespace).DeleteCollection(&v1.DeleteOptions{}, options)
}

func CreateTestCRs(num int, prefix string, ep *v1beta1.ClusterEndpoint, t *testing.T) {
	ep.Labels = map[string]string{"test": prefix}
	for i := 0; i < num; i++ {
		ep.Name = fmt.Sprintf("%s-%d", prefix, i)
		err := CreateCR(ep.DeepCopy())
		if err != nil {
			t.Errorf("%s create failed: %v", ep.Name, err)
			continue
		}
		t.Logf("%s create success.", ep.Name)
	}
}

func DeleteTestCRs(prefix, namespace string, t *testing.T) {
	m := map[string]string{"test": prefix}
	err := DeleteCRs(namespace, v1.ListOptions{
		LabelSelector: labels.FormatLabels(m),
	})
	if err != nil {
		t.Errorf("delete failed: %v", err)
		return
	}
	t.Log("delete success.")
}
