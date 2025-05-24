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

package mapofslices

import (
	"testing"
)

func TestMapOfSlices(t *testing.T) {
	localSchemeBuilder.Test(t).ValidateFixtures()

	// Additional specific tests
	t.Run("empty map", func(t *testing.T) {
		obj := &TestStruct{
			ExtraValues: map[string][]string{},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("nil vs empty slices", func(t *testing.T) {
		obj := &TestStruct{
			ExtraValues: map[string][]string{
				"nil-slice":   nil,
				"empty-slice": {},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValidateFalse(
			"type TestStruct",
			"field TestStruct.ExtraValues",
		)
	})
}
