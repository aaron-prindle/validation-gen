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

package mapofslicesvalidation

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestNestedValidations(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		obj := &TestStruct{
			LimitedExtra: map[string][]string{
				"key1": {"short", "valid"},
				"key2": {"ok"},
			},
			PositiveNumbers: map[string][]int{
				"nums": {1, 2, 3},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("invalid string length", func(t *testing.T) {
		obj := &TestStruct{
			LimitedExtra: map[string][]string{
				"key1": {"this-string-is-too-long"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(field.NewPath("limitedExtra").Key("key1").Index(0), "this-string-is-too-long", "must have at most 10 bytes"),
		)
	})

	t.Run("invalid negative number", func(t *testing.T) {
		obj := &TestStruct{
			PositiveNumbers: map[string][]int{
				"nums": {1, -5, 3},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(field.NewPath("positiveNumbers").Key("nums").Index(1), -5, "must be greater than or equal to 0"),
		)
	})
}
