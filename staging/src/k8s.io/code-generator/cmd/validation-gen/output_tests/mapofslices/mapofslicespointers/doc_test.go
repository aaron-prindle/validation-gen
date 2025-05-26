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

package mapofslicespointers

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestMapOfSlicesPointers(t *testing.T) {
	t.Run("simple struct slices", func(t *testing.T) {
		obj := &TestStruct{
			StructSlices: map[string][]SimpleStruct{
				"users": {
					{Name: "alice", Value: 100},
					{Name: "bob", Value: 50},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("struct field validation - name too long", func(t *testing.T) {
		obj := &TestStruct{
			StructSlices: map[string][]SimpleStruct{
				"invalid": {
					{Name: "this-name-is-way-too-long", Value: 10},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(
				field.NewPath("structSlices").Key("invalid").Index(0).Child("name"),
				"this-name-is-way-too-long",
				"must be no more than 20 bytes",
			).WithOrigin("maxLength"),
		)
	})

	t.Run("struct field validation - negative value", func(t *testing.T) {
		obj := &TestStruct{
			StructSlices: map[string][]SimpleStruct{
				"invalid": {{Name: "test", Value: -5}},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(
				field.NewPath("structSlices").Key("invalid").Index(0).Child("value"),
				-5,
				"must be greater than or equal to 0",
			).WithOrigin("minimum"),
		)
	})

	t.Run("complex struct with pointers", func(t *testing.T) {
		name := "optional"
		num := 42
		obj := &TestStruct{
			ComplexStructSlices: map[string][]ComplexStruct{
				"data": {
					{
						OptionalName: &name,
						OptionalInt:  &num,
						Required:     "required-field",
						Nested: &SimpleStruct{
							Name:  "nested",
							Value: 10,
						},
					},
					{
						// All optional fields nil
						Required: "only-required",
					},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("empty struct slices", func(t *testing.T) {
		obj := &TestStruct{
			StructSlices: map[string][]SimpleStruct{
				"empty": {},
				"nil":   nil,
			},
			ComplexStructSlices: map[string][]ComplexStruct{
				"empty": {},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("primitive slices without pointers", func(t *testing.T) {
		obj := &TestStruct{
			StringSlices: map[string][]string{
				"names": {"alice", "bob", "charlie"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("update operations", func(t *testing.T) {
		oldObj := &TestStruct{
			StructSlices: map[string][]SimpleStruct{
				"users": {{Name: "alice", Value: 100}},
			},
		}
		newObj := &TestStruct{
			StructSlices: map[string][]SimpleStruct{
				"users": {
					{Name: "alice", Value: 100},
					{Name: "bob", Value: 50},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(newObj).OldValue(oldObj).ExpectValid()
	})

	t.Run("all nil pointers in complex struct", func(t *testing.T) {
		obj := &TestStruct{
			ComplexStructSlices: map[string][]ComplexStruct{
				"minimal": {
					{
						OptionalName: nil,
						OptionalInt:  nil,
						Nested:       nil,
						Required:     "required",
					},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("nested struct validation", func(t *testing.T) {
		obj := &TestStruct{
			ComplexStructSlices: map[string][]ComplexStruct{
				"nested": {
					{
						Required: "ok",
						Nested: &SimpleStruct{
							Name:  "this-nested-name-is-too-long",
							Value: 5,
						},
					},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(
				field.NewPath("complexStructSlices").Key("nested").Index(0).Child("nested").Child("name"),
				"this-nested-name-is-too-long",
				"must be no more than 20 bytes",
			).WithOrigin("maxLength"),
		)
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		obj := &TestStruct{
			StructSlices: map[string][]SimpleStruct{
				"errors": {
					{Name: "name-that-is-definitely-too-long", Value: -10},
					{Name: "another-name-that-is-too-long", Value: -5},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(
				field.NewPath("structSlices").Key("errors").Index(0).Child("name"),
				"name-that-is-definitely-too-long",
				"must be no more than 20 bytes",
			).WithOrigin("maxLength"),
			field.Invalid(
				field.NewPath("structSlices").Key("errors").Index(0).Child("value"),
				-10,
				"must be greater than or equal to 0",
			).WithOrigin("minimum"),
			field.Invalid(
				field.NewPath("structSlices").Key("errors").Index(1).Child("name"),
				"another-name-that-is-too-long",
				"must be no more than 20 bytes",
			).WithOrigin("maxLength"),
			field.Invalid(
				field.NewPath("structSlices").Key("errors").Index(1).Child("value"),
				-5,
				"must be greater than or equal to 0",
			).WithOrigin("minimum"),
		)
	})
}
