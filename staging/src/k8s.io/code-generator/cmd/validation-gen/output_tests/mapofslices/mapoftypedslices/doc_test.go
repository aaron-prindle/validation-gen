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

package mapoftypedslices

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestMapOfTypedSlices(t *testing.T) {
	t.Run("valid typed slices", func(t *testing.T) {
		obj := &TestStruct{
			TypedExtra: map[string]ValueList{
				"groups": {"admin", "user"},
				"roles":  {"read", "write", "execute"},
			},
		}
		// Note: ValueList validation won't be called due to signature mismatch
		// Only struct and field validations apply
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedExtra"), obj.TypedExtra, "forced failure: field TestStruct.TypedExtra"),
		)
	})

	t.Run("empty map", func(t *testing.T) {
		obj := &TestStruct{
			TypedExtra: map[string]ValueList{},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedExtra"), obj.TypedExtra, "forced failure: field TestStruct.TypedExtra"),
		)
	})

	t.Run("nil map", func(t *testing.T) {
		obj := &TestStruct{
			TypedExtra: nil,
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedExtra"), obj.TypedExtra, "forced failure: field TestStruct.TypedExtra"),
		)
	})

	t.Run("nil vs empty ValueList", func(t *testing.T) {
		obj := &TestStruct{
			TypedExtra: map[string]ValueList{
				"nil-list":   nil,
				"empty-list": {},
				"with-items": {"item1", "item2"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedExtra"), obj.TypedExtra, "forced failure: field TestStruct.TypedExtra"),
		)
	})

	t.Run("ValueList used directly", func(t *testing.T) {
		// When ValueList is used directly (not as a map value), its validation should work
		vl := ValueList{"test", "values"}
		// This would trigger Validate_ValueList if we had a way to test it directly
		// But in our test framework, we can only test struct types
		_ = vl
	})

	t.Run("update operations - add key", func(t *testing.T) {
		oldObj := &TestStruct{
			TypedExtra: map[string]ValueList{
				"existing": {"value1"},
			},
		}
		newObj := &TestStruct{
			TypedExtra: map[string]ValueList{
				"existing": {"value1"},
				"new":      {"value2", "value3"},
			},
		}
		localSchemeBuilder.Test(t).Value(newObj).OldValue(oldObj).ExpectInvalid(
			field.Invalid(nil, newObj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedExtra"), newObj.TypedExtra, "forced failure: field TestStruct.TypedExtra"),
		)
	})

	t.Run("update operations - modify ValueList", func(t *testing.T) {
		oldObj := &TestStruct{
			TypedExtra: map[string]ValueList{
				"groups": {"admin"},
			},
		}
		newObj := &TestStruct{
			TypedExtra: map[string]ValueList{
				"groups": {"admin", "user", "viewer"},
			},
		}
		localSchemeBuilder.Test(t).Value(newObj).OldValue(oldObj).ExpectInvalid(
			field.Invalid(nil, newObj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedExtra"), newObj.TypedExtra, "forced failure: field TestStruct.TypedExtra"),
		)
	})

	t.Run("no changes on update", func(t *testing.T) {
		obj := &TestStruct{
			TypedExtra: map[string]ValueList{
				"groups": {"admin", "user"},
			},
		}
		// When old and new are the same, ratcheting should skip validation
		localSchemeBuilder.Test(t).Value(obj).OldValue(obj).ExpectValid()
	})

	t.Run("complex scenario", func(t *testing.T) {
		obj := &TestStruct{
			TypedExtra: map[string]ValueList{
				"admins":     {"root", "superuser"},
				"developers": {"alice", "bob", "charlie"},
				"viewers":    {"guest"},
				"empty":      {},
				"nil":        nil,
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedExtra"), obj.TypedExtra, "forced failure: field TestStruct.TypedExtra"),
		)
	})
}
