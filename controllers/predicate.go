// Copyright © 2022 sealos.
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

package controllers

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ResourceChangedPredicate struct {
	predicate.Funcs
}

func (rl *ResourceChangedPredicate) Update(e event.UpdateEvent) bool {
	return true
}

func (rl *ResourceChangedPredicate) Create(e event.CreateEvent) bool {
	return true
}

// Delete returns true if the Delete event should be processed
func (rl *ResourceChangedPredicate) Delete(e event.DeleteEvent) bool {
	return true
}

// Generic returns true if the Generic event should be processed
func (rl *ResourceChangedPredicate) Generic(e event.GenericEvent) bool {
	return true
}
