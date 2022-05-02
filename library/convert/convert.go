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

package convert

import (
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/json"
)

func ResourcesToUnstructuredList(resources []interface{}) ([]*unstructured.Unstructured, error) {
	resourceList := make([]*unstructured.Unstructured, 0)
	if resources == nil || len(resources) == 0 {
		return resourceList, nil
	}

	for _, r := range resources {
		if r != nil {
			value, err := ResourceToUnstructured(r)
			if err != nil {
				return resourceList, nil
			}
			resourceList = append(resourceList, value)
		}
	}
	return resourceList, nil
}

// ConvertResourceToUnstructured func
func ResourceToUnstructured(resource interface{}) (*unstructured.Unstructured, error) {
	unstr := &unstructured.Unstructured{}
	data, err := json.Marshal(resource)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, unstr)
	if err != nil {
		return nil, err
	}
	return unstr, nil
}

func JsonConvert(from interface{}, to interface{}) error {
	var data []byte
	var err error
	if data, err = json.Marshal(from); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(json.Unmarshal(data, to))
}
