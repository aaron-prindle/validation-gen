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

package eachslicevalpointer

import (
	"testing"
)

func TestEachSliceValPointer(t *testing.T) {
	t.Run("complex struct with all nil pointers - EXPECTED TO FAIL", func(t *testing.T) {
		// This test is expected to panic with nil pointer dereference
		// if the generated code doesn't check for nil before calling Validate_SimpleStruct
		obj := &TestStruct{
			ComplexStructSlices: []ComplexStruct{
				{
					OptionalName: nil,
					OptionalInt:  nil,
					Nested:       nil, // This nil pointer should cause the issue
				},
			},
		}
		// This should panic if the bug exists
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})
}
