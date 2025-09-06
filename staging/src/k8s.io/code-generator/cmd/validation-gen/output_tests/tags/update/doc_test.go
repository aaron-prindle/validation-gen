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

	// Test NoModify,NoUnset (set once pattern - like PVC.volumeName)
	unsetStruct := UpdateTestStruct{
		RequiredImmutableField: "required-value", // Always needed for valid structs
	}
	setOnceStruct := UpdateTestStruct{
		SetOnceField:           "initial-value",
		RequiredImmutableField: "required-value",
	}
	modifiedStruct := UpdateTestStruct{
		SetOnceField:           "modified-value",
		RequiredImmutableField: "required-value",
	}

	// Can set from unset to set
	st.Value(&setOnceStruct).OldValue(&unsetStruct).ExpectValid()

	// Cannot modify once set
	st.Value(&modifiedStruct).OldValue(&setOnceStruct).ExpectInvalid(
		field.Forbidden(field.NewPath("setOnceField"), "field cannot be modified once set"),
	)

	// Cannot unset once set
	st.Value(&unsetStruct).OldValue(&setOnceStruct).ExpectInvalid(
		field.Forbidden(field.NewPath("setOnceField"), "field cannot be cleared once set"),
	)

	// Test NoSet (must be set at creation or never)
	createOnlySet := UpdateTestStruct{
		CreateOnlyField:        "value",
		RequiredImmutableField: "required-value",
	}

	// Cannot set after creation if it was unset
	st.Value(&createOnlySet).OldValue(&unsetStruct).ExpectInvalid(
		field.Forbidden(field.NewPath("createOnlyField"), "field cannot be set once created"),
	)

	// Test NoRemoveItem,NoUnset (append only list)
	emptyList := UpdateTestStruct{
		AppendOnlyList:         []string{},
		RequiredImmutableField: "required-value",
	}
	withItems := UpdateTestStruct{
		AppendOnlyList:         []string{"item1", "item2"},
		RequiredImmutableField: "required-value",
	}
	moreItems := UpdateTestStruct{
		AppendOnlyList:         []string{"item1", "item2", "item3"},
		RequiredImmutableField: "required-value",
	}
	fewerItems := UpdateTestStruct{
		AppendOnlyList:         []string{"item1"},
		RequiredImmutableField: "required-value",
	}

	// Can add items
	st.Value(&moreItems).OldValue(&withItems).ExpectValid()

	// Cannot remove items
	st.Value(&fewerItems).OldValue(&withItems).ExpectInvalid(
		field.Forbidden(field.NewPath("appendOnlyList"), "cannot remove items"),
	)

	// Cannot clear list
	st.Value(&emptyList).OldValue(&withItems).ExpectInvalid(
		field.Forbidden(field.NewPath("appendOnlyList"), "cannot remove items"),
	)

	// Test NoUnset (required once set)
	withPointer := UpdateTestStruct{
		RequiredOnceSetField:   ptr.To("value"),
		RequiredImmutableField: "required-value",
	}

	// Can set initially
	st.Value(&withPointer).OldValue(&unsetStruct).ExpectValid()

	// Cannot unset
	st.Value(&unsetStruct).OldValue(&withPointer).ExpectInvalid(
		field.Forbidden(field.NewPath("requiredOnceSetField"), "field cannot be cleared once set"),
	)

	// Test truly immutable field
	// Whatever it has at creation (set or unset) cannot change
	immutableUnset := UpdateTestStruct{
		TrulyImmutableField:    "",
		RequiredImmutableField: "required-value",
	}
	immutableSet := UpdateTestStruct{
		TrulyImmutableField:    "value",
		RequiredImmutableField: "required-value",
	}

	// If created unset, must stay unset (cannot set)
	st.Value(&immutableSet).OldValue(&immutableUnset).ExpectInvalid(
		field.Forbidden(field.NewPath("trulyImmutableField"), "field is immutable"),
	)

	// If created set, must stay set with same value (cannot modify)
	immutableModified := UpdateTestStruct{
		TrulyImmutableField:    "different-value",
		RequiredImmutableField: "required-value",
	}
	st.Value(&immutableModified).OldValue(&immutableSet).ExpectInvalid(
		field.Forbidden(field.NewPath("trulyImmutableField"), "field is immutable"),
	)

	// If created set, cannot be unset
	st.Value(&immutableUnset).OldValue(&immutableSet).ExpectInvalid(
		field.Forbidden(field.NewPath("trulyImmutableField"), "field is immutable"),
	)

	// Test required + immutable
	// Must be set at creation and cannot change
	requiredImmutableDifferent := UpdateTestStruct{
		RequiredImmutableField: "different-value",
	}

	// Cannot modify required immutable field
	st.Value(&requiredImmutableDifferent).OldValue(&unsetStruct).ExpectInvalid(
		field.Forbidden(field.NewPath("requiredImmutableField"), "field is immutable"),
	)

	// Test NoAddItem,NoRemoveItem (immutable list)
	immutableListWithItems := UpdateTestStruct{
		ImmutableList: []Item{
			{Name: "item1", Value: "value1"},
			{Name: "item2", Value: "value2"},
		},
		RequiredImmutableField: "required-value",
	}
	immutableListMoreItems := UpdateTestStruct{
		ImmutableList: []Item{
			{Name: "item1", Value: "value1"},
			{Name: "item2", Value: "value2"},
			{Name: "item3", Value: "value3"},
		},
		RequiredImmutableField: "required-value",
	}
	immutableListFewerItems := UpdateTestStruct{
		ImmutableList: []Item{
			{Name: "item1", Value: "value1"},
		},
		RequiredImmutableField: "required-value",
	}

	// Cannot add items
	st.Value(&immutableListMoreItems).OldValue(&immutableListWithItems).ExpectInvalid(
		field.Forbidden(field.NewPath("immutableList").Index(2), "cannot add new items"),
	)

	// Cannot remove items
	st.Value(&immutableListFewerItems).OldValue(&immutableListWithItems).ExpectInvalid(
		field.Forbidden(field.NewPath("immutableList"), "cannot remove items"),
	)

	// Test NoRemoveItem for map
	mapEmpty := UpdateTestStruct{
		NoRemoveMap:            map[string]string{},
		RequiredImmutableField: "required-value",
	}
	mapWithEntries := UpdateTestStruct{
		NoRemoveMap: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		RequiredImmutableField: "required-value",
	}
	mapMoreEntries := UpdateTestStruct{
		NoRemoveMap: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
		RequiredImmutableField: "required-value",
	}
	mapFewerEntries := UpdateTestStruct{
		NoRemoveMap: map[string]string{
			"key1": "value1",
		},
		RequiredImmutableField: "required-value",
	}

	// Can add entries
	st.Value(&mapMoreEntries).OldValue(&mapWithEntries).ExpectValid()

	// Cannot remove entries
	st.Value(&mapFewerEntries).OldValue(&mapWithEntries).ExpectInvalid(
		field.Forbidden(field.NewPath("noRemoveMap").Key("key2"), "cannot remove entries"),
	)

	// Can clear if empty (no entries to remove)
	st.Value(&mapEmpty).OldValue(&mapEmpty).ExpectValid()

	// Cannot clear if has entries (would remove them)
	st.Value(&mapEmpty).OldValue(&mapWithEntries).ExpectInvalid(
		field.Forbidden(field.NewPath("noRemoveMap").Key("key1"), "cannot remove entries"),
	)
}
