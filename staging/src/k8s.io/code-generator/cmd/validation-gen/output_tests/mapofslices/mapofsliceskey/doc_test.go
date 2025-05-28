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

package mapofsliceskey

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestMapOfSlicesKey(t *testing.T) {
	t.Run("basic key validation", func(t *testing.T) {
		obj := &TestStruct{
			BasicKeys: map[string][]string{
				"key1": {"value1", "value2"},
				"key2": {"val"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("basicKeys"), "key1", "forced failure: key BasicKeys"),
			field.Invalid(field.NewPath("basicKeys"), "key2", "forced failure: key BasicKeys"),
		)
	})

	t.Run("multiple key validations", func(t *testing.T) {
		obj := &TestStruct{
			MultipleKeys: map[string][]string{
				"key": {"value"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("multipleKeys"), "key", "forced failure: key MultipleKeys #1"),
			field.Invalid(field.NewPath("multipleKeys"), "key", "forced failure: key MultipleKeys #2"),
		)
	})

	t.Run("combined key and slice size validation", func(t *testing.T) {
		obj := &TestStruct{
			KeysAndSlices: map[string][]string{
				"key1": {"one", "two", "three"},
				"key2": {"one", "two", "three", "four"}, // exceeds maxItems
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("keysAndSlices"), "key1", "forced failure: key KeysAndSlices"),
			field.Invalid(field.NewPath("keysAndSlices"), "key2", "forced failure: key KeysAndSlices"),
			field.TooMany(field.NewPath("keysAndSlices").Key("key2"), 4, 3),
		)
	})

	t.Run("key and element validation", func(t *testing.T) {
		obj := &TestStruct{
			KeysAndElements: map[string][]string{
				"key1": {"elem1", "elem2"},
				"key2": {"elem3"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("keysAndElements"), "key1", "forced failure: key KeysAndElements"),
			field.Invalid(field.NewPath("keysAndElements"), "key2", "forced failure: key KeysAndElements"),
			field.Invalid(field.NewPath("keysAndElements").Key("key1").Index(0), "elem1", "forced failure: element KeysAndElements"),
			field.Invalid(field.NewPath("keysAndElements").Key("key1").Index(1), "elem2", "forced failure: element KeysAndElements"),
			field.Invalid(field.NewPath("keysAndElements").Key("key2").Index(0), "elem3", "forced failure: element KeysAndElements"),
		)
	})

	t.Run("integer slice with key validation", func(t *testing.T) {
		obj := &TestStruct{
			IntSliceKeys: map[string][]int{
				"nums": {1, 2, 3},
				"data": {0, 100},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("intSliceKeys"), "data", "forced failure: key IntSliceKeys"),
			field.Invalid(field.NewPath("intSliceKeys"), "nums", "forced failure: key IntSliceKeys"),
			field.Invalid(field.NewPath("intSliceKeys").Key("data").Index(0), 0, "forced failure: element IntSliceKeys"),
			field.Invalid(field.NewPath("intSliceKeys").Key("data").Index(1), 100, "forced failure: element IntSliceKeys"),
			field.Invalid(field.NewPath("intSliceKeys").Key("nums").Index(0), 1, "forced failure: element IntSliceKeys"),
			field.Invalid(field.NewPath("intSliceKeys").Key("nums").Index(1), 2, "forced failure: element IntSliceKeys"),
			field.Invalid(field.NewPath("intSliceKeys").Key("nums").Index(2), 3, "forced failure: element IntSliceKeys"),
		)
	})

	t.Run("struct slices with key validation", func(t *testing.T) {
		obj := &TestStruct{
			StructSliceKeys: map[string][]User{
				"users":  {{Name: "alice", ID: 1}, {Name: "bob", ID: 2}},
				"admins": {{Name: "root", ID: 0}},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("structSliceKeys"), "admins", "forced failure: key StructSliceKeys"),
			field.Invalid(field.NewPath("structSliceKeys"), "users", "forced failure: key StructSliceKeys"),
			field.Invalid(field.NewPath("structSliceKeys").Key("admins").Index(0), User{Name: "root", ID: 0}, "forced failure: element StructSliceKeys"),
			field.Invalid(field.NewPath("structSliceKeys").Key("users").Index(0), User{Name: "alice", ID: 1}, "forced failure: element StructSliceKeys"),
			field.Invalid(field.NewPath("structSliceKeys").Key("users").Index(1), User{Name: "bob", ID: 2}, "forced failure: element StructSliceKeys"),
			field.Invalid(nil, User{Name: "root", ID: 0}, "forced failure: type User"),
			field.Invalid(nil, User{Name: "alice", ID: 1}, "forced failure: type User"),
			field.Invalid(nil, User{Name: "bob", ID: 2}, "forced failure: type User"),
		)
	})

	t.Run("struct slices exceeding max items", func(t *testing.T) {
		obj := &TestStruct{
			StructSliceKeys: map[string][]User{
				"toomany": make([]User, 6), // exceeds maxItems of 5
			},
		}
		// Initialize the slice with valid data
		for i := 0; i < 6; i++ {
			obj.StructSliceKeys["toomany"][i] = User{Name: "user", ID: i}
		}

		// Build expected errors
		expectedErrors := []*field.Error{
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("structSliceKeys"), "toomany", "forced failure: key StructSliceKeys"),
			field.TooMany(field.NewPath("structSliceKeys").Key("toomany"), 6, 5),
		}
		// Add element validation errors for each user
		for i := 0; i < 6; i++ {
			expectedErrors = append(expectedErrors,
				field.Invalid(field.NewPath("structSliceKeys").Key("toomany").Index(i),
					obj.StructSliceKeys["toomany"][i], "forced failure: element StructSliceKeys"),
			)
		}
		// Add type validation errors for each user
		for i := 0; i < 6; i++ {
			expectedErrors = append(expectedErrors,
				field.Invalid(nil, obj.StructSliceKeys["toomany"][i], "forced failure: type User"),
			)
		}

		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(expectedErrors...)
	})

	t.Run("typedef values with key validation", func(t *testing.T) {
		obj := &TestStruct{
			TypedefKeys: map[string]ValueList{
				"short": {"val1", "val2"},
				"key":   {"value"},
			},
		}
		// Note: ValueList validation won't be called due to typedef limitation
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedefKeys"), "key", "forced failure: key TypedefKeys"),
			field.Invalid(field.NewPath("typedefKeys"), "short", "forced failure: key TypedefKeys"),
			field.Invalid(field.NewPath("typedefKeys"), obj.TypedefKeys, "forced failure: field TestStruct.TypedefKeys"),
		)
	})

	t.Run("empty maps", func(t *testing.T) {
		obj := &TestStruct{
			BasicKeys:       map[string][]string{},
			MultipleKeys:    map[string][]string{},
			KeysAndSlices:   map[string][]string{},
			KeysAndElements: map[string][]string{},
			IntSliceKeys:    map[string][]int{},
			StructSliceKeys: map[string][]User{},
			TypedefKeys:     map[string]ValueList{},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedefKeys"), obj.TypedefKeys, "forced failure: field TestStruct.TypedefKeys"),
		)
	})

	t.Run("nil maps", func(t *testing.T) {
		obj := &TestStruct{
			BasicKeys:     nil,
			MultipleKeys:  nil,
			KeysAndSlices: nil,
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedefKeys"), obj.TypedefKeys, "forced failure: field TestStruct.TypedefKeys"),
		)
	})

	t.Run("unvalidated map remains unvalidated", func(t *testing.T) {
		obj := &TestStruct{
			UnvalidatedMap: map[string][]string{
				"any-key": {"any", "values"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("typedefKeys"), obj.TypedefKeys, "forced failure: field TestStruct.TypedefKeys"),
		)
	})

	t.Run("update operations", func(t *testing.T) {
		oldObj := &TestStruct{
			BasicKeys: map[string][]string{
				"old": {"value"},
			},
		}
		newObj := &TestStruct{
			BasicKeys: map[string][]string{
				"old": {"value"},
				"new": {"value2"},
			},
		}
		localSchemeBuilder.Test(t).Value(newObj).OldValue(oldObj).ExpectInvalid(
			field.Invalid(nil, newObj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("basicKeys"), "new", "forced failure: key BasicKeys"),
			field.Invalid(field.NewPath("basicKeys"), "old", "forced failure: key BasicKeys"),
		)
	})

	t.Run("all validations together", func(t *testing.T) {
		obj := &TestStruct{
			BasicKeys: map[string][]string{
				"key": {"val"},
			},
			KeysAndSlices: map[string][]string{
				"short": {"one", "two"}, // within maxItems limit
			},
			IntSliceKeys: map[string][]int{
				"nums": {1, 2},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(nil, obj, "forced failure: type TestStruct"),
			field.Invalid(field.NewPath("basicKeys"), "key", "forced failure: key BasicKeys"),
			field.Invalid(field.NewPath("keysAndSlices"), "short", "forced failure: key KeysAndSlices"),
			field.Invalid(field.NewPath("intSliceKeys"), "nums", "forced failure: key IntSliceKeys"),
			field.Invalid(field.NewPath("intSliceKeys").Key("nums").Index(0), 1, "forced failure: element IntSliceKeys"),
			field.Invalid(field.NewPath("intSliceKeys").Key("nums").Index(1), 2, "forced failure: element IntSliceKeys"),
			field.Invalid(field.NewPath("typedefKeys"), obj.TypedefKeys, "forced failure: field TestStruct.TypedefKeys"),
		)
	})
}
