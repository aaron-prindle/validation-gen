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

package mapofcomplexslices

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestMapOfComplexSlices(t *testing.T) {
	localSchemeBuilder.Test(t).ValidateFixtures()

	t.Run("valid complex values", func(t *testing.T) {
		obj := &TestStruct{
			ComplexMap: map[string][]ComplexValue{
				"users": {
					{Name: "alice", Value: 100},
					{Name: "bob", Value: 50},
				},
				"admins": {
					{Name: "root", Value: 0},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("validateFalse on elements", func(t *testing.T) {
		obj := &TestStruct{
			ComplexMap: map[string][]ComplexValue{
				"test": {{Name: "test", Value: 1}},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValidateFalse(
			"type TestStruct",
			"type ComplexValue",
			"element",
		)
	})

	t.Run("name too long", func(t *testing.T) {
		obj := &TestStruct{
			ComplexMap: map[string][]ComplexValue{
				"users": {
					{
						Name:  "this-name-is-way-too-long-and-exceeds-the-fifty-character-limit",
						Value: 10,
					},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(
				field.NewPath("complexMap").Key("users").Index(0).Child("name"),
				"this-name-is-way-too-long-and-exceeds-the-fifty-character-limit",
				"must have at most 50 bytes",
			),
		)
	})

	t.Run("negative value", func(t *testing.T) {
		obj := &TestStruct{
			ComplexMap: map[string][]ComplexValue{
				"users": {
					{Name: "alice", Value: -5},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(
				field.NewPath("complexMap").Key("users").Index(0).Child("value"),
				-5,
				"must be greater than or equal to 0",
			),
		)
	})

	t.Run("empty map", func(t *testing.T) {
		obj := &TestStruct{
			ComplexMap: map[string][]ComplexValue{},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValidateFalse("type TestStruct")
	})

	t.Run("nil slice values", func(t *testing.T) {
		obj := &TestStruct{
			ComplexMap: map[string][]ComplexValue{
				"nil-slice":   nil,
				"empty-slice": {},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValidateFalse("type TestStruct")
	})

	t.Run("update operations", func(t *testing.T) {
		oldObj := &TestStruct{
			ComplexMap: map[string][]ComplexValue{
				"users": {
					{Name: "alice", Value: 100},
				},
			},
		}
		newObj := &TestStruct{
			ComplexMap: map[string][]ComplexValue{
				"users": {
					{Name: "alice", Value: 100},
					{Name: "bob", Value: 50},
				},
				"admins": {
					{Name: "root", Value: 0},
				},
			},
		}
		// Should validate both old and new values, including the validateFalse tags
		localSchemeBuilder.Test(t).Value(newObj).OldValue(oldObj).ExpectValidateFalse(
			"type TestStruct",
			"type ComplexValue",
			"type ComplexValue",
			"type ComplexValue",
			"element",
			"element",
			"element",
		)
	})

	t.Run("multiple validation errors", func(t *testing.T) {
		obj := &TestStruct{
			ComplexMap: map[string][]ComplexValue{
				"users": {
					{
						Name:  "this-name-is-way-too-long-and-exceeds-the-fifty-character-limit",
						Value: -10,
					},
					{
						Name:  "another-name-that-is-also-too-long-and-exceeds-the-limit",
						Value: -5,
					},
				},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectMatches(
			field.NewDefaultErrorMatcher(),
			field.ErrorList{
				field.Invalid(
					field.NewPath("complexMap").Key("users").Index(0).Child("name"),
					"this-name-is-way-too-long-and-exceeds-the-fifty-character-limit",
					"must have at most 50 bytes",
				),
				field.Invalid(
					field.NewPath("complexMap").Key("users").Index(0).Child("value"),
					-10,
					"must be greater than or equal to 0",
				),
				field.Invalid(
					field.NewPath("complexMap").Key("users").Index(1).Child("name"),
					"another-name-that-is-also-too-long-and-exceeds-the-limit",
					"must have at most 50 bytes",
				),
				field.Invalid(
					field.NewPath("complexMap").Key("users").Index(1).Child("value"),
					-5,
					"must be greater than or equal to 0",
				),
			},
		)
	})
}
