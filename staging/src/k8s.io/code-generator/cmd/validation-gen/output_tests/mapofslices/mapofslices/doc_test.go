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

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestMapOfSlices(t *testing.T) {
	t.Run("valid values", func(t *testing.T) {
		obj := &TestStruct{
			ExtraValues: map[string][]string{
				"groups":  {"admin", "user"},
				"regions": {"us-east", "us-west"},
			},
		}
		// This should trigger validateFalse
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("extraValues"), obj.ExtraValues, "forced failure: field TestStruct.ExtraValues"),
		)
	})

	t.Run("empty map", func(t *testing.T) {
		obj := &TestStruct{
			ExtraValues: map[string][]string{},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("extraValues"), obj.ExtraValues, "forced failure: field TestStruct.ExtraValues"),
		)
	})

	t.Run("nil map", func(t *testing.T) {
		obj := &TestStruct{
			ExtraValues: nil,
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("extraValues"), obj.ExtraValues, "forced failure: field TestStruct.ExtraValues"),
		)
	})

	t.Run("nil vs empty slices", func(t *testing.T) {
		obj := &TestStruct{
			ExtraValues: map[string][]string{
				"nil-slice":   nil,
				"empty-slice": {},
				"with-values": {"val1", "val2"},
			},
		}
		// Should trigger validateFalse for the struct and field
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("extraValues"), obj.ExtraValues, "forced failure: field TestStruct.ExtraValues"),
		)
	})

	t.Run("large map", func(t *testing.T) {
		obj := &TestStruct{
			ExtraValues: map[string][]string{
				"key1": {"a", "b", "c"},
				"key2": {"d", "e", "f"},
				"key3": {"g", "h", "i"},
				"key4": {"j", "k", "l"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("extraValues"), obj.ExtraValues, "forced failure: field TestStruct.ExtraValues"),
		)
	})

	t.Run("update operations - add key", func(t *testing.T) {
		oldObj := &TestStruct{
			ExtraValues: map[string][]string{
				"groups": {"admin"},
			},
		}
		newObj := &TestStruct{
			ExtraValues: map[string][]string{
				"groups": {"admin", "user"},
				"roles":  {"read"},
			},
		}
		localSchemeBuilder.Test(t).Value(newObj).OldValue(oldObj).ExpectInvalid(
			field.Invalid(nil, newObj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("extraValues"), newObj.ExtraValues, "forced failure: field TestStruct.ExtraValues"),
		)
	})

	t.Run("update operations - remove key", func(t *testing.T) {
		oldObj := &TestStruct{
			ExtraValues: map[string][]string{
				"groups": {"admin", "user"},
				"roles":  {"read", "write"},
			},
		}
		newObj := &TestStruct{
			ExtraValues: map[string][]string{
				"groups": {"admin", "user"},
			},
		}
		localSchemeBuilder.Test(t).Value(newObj).OldValue(oldObj).ExpectInvalid(
			field.Invalid(nil, newObj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("extraValues"), newObj.ExtraValues, "forced failure: field TestStruct.ExtraValues"),
		)
	})

	t.Run("update operations - modify slice", func(t *testing.T) {
		oldObj := &TestStruct{
			ExtraValues: map[string][]string{
				"groups": {"admin"},
			},
		}
		newObj := &TestStruct{
			ExtraValues: map[string][]string{
				"groups": {"admin", "user", "viewer"},
			},
		}
		localSchemeBuilder.Test(t).Value(newObj).OldValue(oldObj).ExpectInvalid(
			field.Invalid(nil, newObj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("extraValues"), newObj.ExtraValues, "forced failure: field TestStruct.ExtraValues"),
		)
	})

	t.Run("unvalidated field remains unvalidated", func(t *testing.T) {
		obj := &TestStruct{
			ExtraValues: map[string][]string{
				"test": {"value"},
			},
			UnvalidatedMapOfSlices: map[string][]int{
				"numbers": {1, 2, 3, 4, 5},
			},
		}
		// Only the validated fields should trigger errors
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("extraValues"), obj.ExtraValues, "forced failure: field TestStruct.ExtraValues"),
		)
	})
}
