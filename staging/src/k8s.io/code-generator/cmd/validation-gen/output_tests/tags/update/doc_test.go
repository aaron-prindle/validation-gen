/*
Copyright 2025 The Kubernetes Authors.

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

package update

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

func TestUpdateTags(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// Base struct with all required fields (including those with +optional +default)
	baseStruct := UpdateTestStruct{
		PrimitiveRequiredNoModify:      "required-value",
		OptionalDefaultNoModify:        "default-value",
		OptionalDefaultNoSet:           "default-value",
		OptionalDefaultNoUnset:         "default-value",
		OptionalDefaultIntNoModify:     42,
		OptionalDefaultBoolNoModify:    true,
		OptionalDefaultPointerNoModify: ptr.To("pointer-default"),
	}

	// Test Primitive Fields (non-pointer)
	t.Run("Primitive NoSet - UPDATE(set) prevented", func(t *testing.T) {
		// Start with empty/unset value
		old := baseStruct
		old.PrimitiveNoSet = "" // unset

		// Try to set it
		new := baseStruct
		new.PrimitiveNoSet = "value"

		// Cannot set after creation (unset to set transition)
		st.Value(&new).OldValue(&old).ExpectInvalid(
			field.Forbidden(field.NewPath("primitiveNoSet"), "field cannot be set once created"),
		)

		// Can remain unset
		st.Value(&old).OldValue(&old).ExpectValid()
	})

	t.Run("Primitive NoUnset - UPDATE(unset) prevented", func(t *testing.T) {
		oldWithValue := baseStruct
		oldWithValue.PrimitiveNoUnset = "value"

		newWithValue := baseStruct
		newWithValue.PrimitiveNoUnset = "value"

		newUnset := baseStruct
		newUnset.PrimitiveNoUnset = ""

		// Can set initially (empty to non-empty)
		st.Value(&oldWithValue).OldValue(&baseStruct).ExpectValid()

		// Cannot unset (non-empty to empty)
		st.Value(&newUnset).OldValue(&oldWithValue).ExpectInvalid(
			field.Forbidden(field.NewPath("primitiveNoUnset"), "field cannot be cleared once set"),
		)

		// Can keep the same value
		st.Value(&newWithValue).OldValue(&oldWithValue).ExpectValid()
	})

	t.Run("Primitive NoModify - UPDATE(modify) prevented", func(t *testing.T) {
		oldEmpty := baseStruct
		oldEmpty.PrimitiveNoModify = ""

		withValue := baseStruct
		withValue.PrimitiveNoModify = "value"

		modified := baseStruct
		modified.PrimitiveNoModify = "different"

		// Can set initially (UPDATE(set) allowed - empty to non-empty)
		st.Value(&withValue).OldValue(&oldEmpty).ExpectValid()

		// Can unset (UPDATE(unset) allowed - non-empty to empty)
		st.Value(&oldEmpty).OldValue(&withValue).ExpectValid()

		// Cannot modify (non-empty to different non-empty)
		st.Value(&modified).OldValue(&withValue).ExpectInvalid(
			field.Forbidden(field.NewPath("primitiveNoModify"), "field cannot be modified once set"),
		)
	})

	t.Run("Primitive Fully Restricted", func(t *testing.T) {
		oldEmpty := baseStruct
		oldEmpty.PrimitiveFullyRestricted = ""

		withValue := baseStruct
		withValue.PrimitiveFullyRestricted = "value"

		modified := baseStruct
		modified.PrimitiveFullyRestricted = "different"

		// Cannot set (NoSet)
		st.Value(&withValue).OldValue(&oldEmpty).ExpectInvalid(
			field.Forbidden(field.NewPath("primitiveFullyRestricted"), "field cannot be set once created"),
		)

		// If somehow set, cannot unset (NoUnset)
		st.Value(&oldEmpty).OldValue(&withValue).ExpectInvalid(
			field.Forbidden(field.NewPath("primitiveFullyRestricted"), "field cannot be cleared once set"),
		)

		// If somehow set, cannot modify (NoModify)
		st.Value(&modified).OldValue(&withValue).ExpectInvalid(
			field.Forbidden(field.NewPath("primitiveFullyRestricted"), "field cannot be modified once set"),
		)
	})

	t.Run("Set-Once Pattern", func(t *testing.T) {
		oldEmpty := baseStruct
		oldEmpty.PrimitiveSetOnce = ""

		withValue := baseStruct
		withValue.PrimitiveSetOnce = "value"

		modified := baseStruct
		modified.PrimitiveSetOnce = "different"

		// Can set once (empty to non-empty)
		st.Value(&withValue).OldValue(&oldEmpty).ExpectValid()

		// Cannot unset (NoUnset)
		st.Value(&oldEmpty).OldValue(&withValue).ExpectInvalid(
			field.Forbidden(field.NewPath("primitiveSetOnce"), "field cannot be cleared once set"),
		)

		// Cannot modify (NoModify)
		st.Value(&modified).OldValue(&withValue).ExpectInvalid(
			field.Forbidden(field.NewPath("primitiveSetOnce"), "field cannot be modified once set"),
		)
	})

	// Test different primitive types
	t.Run("Int NoModify", func(t *testing.T) {
		// For non-pointer int, zero is considered "unset" per our isUnset logic
		oldZero := baseStruct
		oldZero.IntNoModify = 0

		withValue := baseStruct
		withValue.IntNoModify = 10

		// Can transition from 0 to 10 (unset to set is allowed by NoModify)
		st.Value(&withValue).OldValue(&oldZero).ExpectValid()

		// Cannot modify from one non-zero to another
		modified := baseStruct
		modified.IntNoModify = 20
		st.Value(&modified).OldValue(&withValue).ExpectInvalid(
			field.Forbidden(field.NewPath("intNoModify"), "field cannot be modified once set"),
		)
	})

	t.Run("Bool NoModify", func(t *testing.T) {
		// For non-pointer bool, false is considered "unset" per our isUnset logic
		oldFalse := baseStruct
		oldFalse.BoolNoModify = false

		withTrue := baseStruct
		withTrue.BoolNoModify = true

		// Can transition from false to true (unset to set is allowed by NoModify)
		st.Value(&withTrue).OldValue(&oldFalse).ExpectValid()
	})

	t.Run("Float64 NoModify", func(t *testing.T) {
		// For non-pointer float64, 0.0 is considered "unset" per our isUnset logic
		oldZero := baseStruct
		oldZero.Float64NoModify = 0.0

		withValue := baseStruct
		withValue.Float64NoModify = 3.14

		// Can transition from 0.0 to 3.14 (unset to set is allowed by NoModify)
		st.Value(&withValue).OldValue(&oldZero).ExpectValid()
	})

	// Test Optional + Default Fields
	t.Run("Optional+Default NoModify", func(t *testing.T) {
		// Cannot modify from default
		modified := baseStruct
		modified.OptionalDefaultNoModify = "different"
		st.Value(&modified).OldValue(&baseStruct).ExpectInvalid(
			field.Forbidden(field.NewPath("optionalDefaultNoModify"), "field cannot be modified once set"),
		)
	})

	t.Run("Optional+Default NoSet", func(t *testing.T) {
		// Field is required due to default, so it's always set
		// Can keep default value
		st.Value(&baseStruct).OldValue(&baseStruct).ExpectValid()

		// Can change it (NoSet only prevents unset→set transitions)
		modified := baseStruct
		modified.OptionalDefaultNoSet = "different"
		st.Value(&modified).OldValue(&baseStruct).ExpectValid()
	})

	t.Run("Optional+Default NoUnset", func(t *testing.T) {
		// Can change to different value
		modified := baseStruct
		modified.OptionalDefaultNoUnset = "different"
		st.Value(&modified).OldValue(&baseStruct).ExpectValid()
	})

	t.Run("Optional+Default Int NoModify", func(t *testing.T) {
		// Cannot modify from default (42)
		modified := baseStruct
		modified.OptionalDefaultIntNoModify = 100
		st.Value(&modified).OldValue(&baseStruct).ExpectInvalid(
			field.Forbidden(field.NewPath("optionalDefaultIntNoModify"), "field cannot be modified once set"),
		)
	})

	t.Run("Optional+Default Bool NoModify", func(t *testing.T) {
		// The default is true, cannot change it
	})

	t.Run("Optional+Default Pointer NoModify", func(t *testing.T) {
		// Cannot modify from default
		modified := baseStruct
		modified.OptionalDefaultPointerNoModify = ptr.To("different")
		st.Value(&modified).OldValue(&baseStruct).ExpectInvalid(
			field.Forbidden(field.NewPath("optionalDefaultPointerNoModify"), "field cannot be modified once set"),
		)
	})

	// Test Struct Fields
	t.Run("Struct NoModify", func(t *testing.T) {
		withStruct := baseStruct
		withStruct.StructNoModify = TestStruct{StringField: "value"}

		modifiedStruct := baseStruct
		modifiedStruct.StructNoModify = TestStruct{StringField: "different"}

		// Cannot modify (struct fields are always set, never unset)
		st.Value(&modifiedStruct).OldValue(&withStruct).ExpectInvalid(
			field.Forbidden(field.NewPath("structNoModify"), "field cannot be modified once set"),
		)
	})

	// Test Pointer Fields
	t.Run("Pointer NoSet", func(t *testing.T) {
		withSet := baseStruct
		withSet.PointerNoSet = ptr.To("value")

		// Cannot set after creation (nil to non-nil)
		st.Value(&withSet).OldValue(&baseStruct).ExpectInvalid(
			field.Forbidden(field.NewPath("pointerNoSet"), "field cannot be set once created"),
		)
	})

	t.Run("Pointer NoUnset", func(t *testing.T) {
		withPointer := baseStruct
		withPointer.PointerNoUnset = ptr.To("value")

		// Can set initially (nil to non-nil)
		st.Value(&withPointer).OldValue(&baseStruct).ExpectValid()

		// Cannot unset (non-nil to nil)
		st.Value(&baseStruct).OldValue(&withPointer).ExpectInvalid(
			field.Forbidden(field.NewPath("pointerNoUnset"), "field cannot be cleared once set"),
		)
	})

	t.Run("Pointer NoModify", func(t *testing.T) {
		withPointer := baseStruct
		withPointer.PointerNoModify = ptr.To("value")

		modifiedPointer := baseStruct
		modifiedPointer.PointerNoModify = ptr.To("different")

		// Can set initially
		st.Value(&withPointer).OldValue(&baseStruct).ExpectValid()

		// Can unset (NoModify allows set/unset transitions)
		st.Value(&baseStruct).OldValue(&withPointer).ExpectValid()

		// Cannot modify content
		st.Value(&modifiedPointer).OldValue(&withPointer).ExpectInvalid(
			field.Forbidden(field.NewPath("pointerNoModify"), "field cannot be modified once set"),
		)
	})

	// Lists and maps - ensure no update validators are generated
	t.Run("Lists and maps have no update constraints", func(t *testing.T) {
		// Lists and maps can be freely modified
		withList := baseStruct
		withList.ListField = []string{"item1", "item2"}

		modifiedList := baseStruct
		modifiedList.ListField = []string{"item3", "item4"}

		withMap := baseStruct
		withMap.MapField = map[string]string{"key1": "value1"}

		modifiedMap := baseStruct
		modifiedMap.MapField = map[string]string{"key2": "value2"}

		// All modifications should be valid - no update constraints on lists/maps
		st.Value(&withList).OldValue(&baseStruct).ExpectValid()
		st.Value(&modifiedList).OldValue(&withList).ExpectValid()
		st.Value(&baseStruct).OldValue(&withList).ExpectValid()

		st.Value(&withMap).OldValue(&baseStruct).ExpectValid()
		st.Value(&modifiedMap).OldValue(&withMap).ExpectValid()
		st.Value(&baseStruct).OldValue(&withMap).ExpectValid()
	})
}
