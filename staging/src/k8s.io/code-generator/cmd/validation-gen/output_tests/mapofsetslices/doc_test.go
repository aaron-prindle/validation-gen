/*
Copyright 2024 The Kubernetes Authors.

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

package mapofsetslices

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestMapOfSetSlices(t *testing.T) {
	t.Run("unique values", func(t *testing.T) {
		obj := &TestStruct{
			UniqueValues: map[string][]string{
				"groups": {"admin", "user", "viewer"},
				"roles":  {"read", "write"},
			},
			UniqueInts: map[string][]int{
				"ports": {80, 443, 8080},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("duplicate strings", func(t *testing.T) {
		obj := &TestStruct{
			UniqueValues: map[string][]string{
				"groups": {"admin", "user", "admin"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Duplicate(field.NewPath("uniqueValues").Key("groups").Index(2), "admin"),
		)
	})

	t.Run("duplicate integers", func(t *testing.T) {
		obj := &TestStruct{
			UniqueInts: map[string][]int{
				"ports": {80, 443, 80},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Duplicate(field.NewPath("uniqueInts").Key("ports").Index(2), 80),
		)
	})
}
