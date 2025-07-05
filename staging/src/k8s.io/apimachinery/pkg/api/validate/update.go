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

package validate

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// NoSet verifies that a field cannot be set (transition from unset to set).
func NoSet[T any](_ context.Context, op operation.Operation, fldPath *field.Path, value, oldValue T) field.ErrorList {
	if op.Type != operation.Update {
		return nil
	}

	// Use reflection to properly handle the generic type
	valueIsUnset := isUnsetValue(value)
	oldValueIsUnset := isUnsetValue(oldValue)

	if oldValueIsUnset && !valueIsUnset {
		return field.ErrorList{
			field.Forbidden(fldPath, "field cannot be set once created"),
		}
	}

	return nil
}

// NoUnset verifies that a field cannot be unset (transition from set to unset).
func NoUnset[T any](_ context.Context, op operation.Operation, fldPath *field.Path, value, oldValue T) field.ErrorList {
	if op.Type != operation.Update {
		return nil
	}

	// Special handling for string values since the generated code passes pointers to strings
	switch v := any(value).(type) {
	case *string:
		o := any(oldValue).(*string)
		// If old value was set (non-empty) and new value is unset (empty)
		if o != nil && *o != "" && (v == nil || *v == "") {
			return field.ErrorList{
				field.Forbidden(fldPath, "field cannot be cleared once set"),
			}
		}
		return nil
	case **string:
		// Handle double pointer case for pointer fields
		o := any(oldValue).(**string)
		if o != nil && *o != nil && **o != "" && (v == nil || *v == nil || **v == "") {
			return field.ErrorList{
				field.Forbidden(fldPath, "field cannot be cleared once set"),
			}
		}
		return nil
	}

	// For other types, use the generic logic
	valueIsUnset := isUnsetValue(value)
	oldValueIsUnset := isUnsetValue(oldValue)

	if !oldValueIsUnset && valueIsUnset {
		return field.ErrorList{
			field.Forbidden(fldPath, "field cannot be cleared once set"),
		}
	}

	return nil
}

// NoModify verifies that a field's value cannot be modified (but allows set/unset transitions).
func NoModify[T any](_ context.Context, op operation.Operation, fldPath *field.Path, value, oldValue T) field.ErrorList {
	if op.Type != operation.Update {
		return nil
	}

	// Special handling for string values since the generated code passes pointers to strings
	switch v := any(value).(type) {
	case *string:
		o := any(oldValue).(*string)
		// Allow transition from empty to non-empty (set)
		if o != nil && *o == "" && v != nil && *v != "" {
			return nil
		}
		// Allow transition from non-empty to empty (unset)
		if o != nil && *o != "" && v != nil && *v == "" {
			return nil
		}
		// If both are set with values, check if they're equal
		if o != nil && v != nil && *o != "" && *v != "" && *o != *v {
			return field.ErrorList{
				field.Forbidden(fldPath, "field cannot be modified once set"),
			}
		}
		return nil
	case **string:
		// Handle double pointer case
		o := any(oldValue).(**string)
		if o != nil && *o != nil && v != nil && *v != nil {
			// Allow set/unset transitions
			if **o == "" && **v != "" {
				return nil
			}
			if **o != "" && **v == "" {
				return nil
			}
			// If both are set with values, check if they're equal
			if **o != "" && **v != "" && **o != **v {
				return field.ErrorList{
					field.Forbidden(fldPath, "field cannot be modified once set"),
				}
			}
		}
		return nil
	}

	// For other types, use the generic logic
	valueIsUnset := isUnsetValue(value)
	oldValueIsUnset := isUnsetValue(oldValue)

	// Allow transitions between set/unset
	if oldValueIsUnset || valueIsUnset {
		return nil
	}

	// Both are set - check if they're equal
	if !equality.Semantic.DeepEqual(value, oldValue) {
		return field.ErrorList{
			field.Forbidden(fldPath, "field cannot be modified once set"),
		}
	}

	return nil
}

// NoAddItem verifies that items cannot be added to a list.
// The optional keyFields parameter specifies the fields to use for list map keys.
func NoAddItem[T any](_ context.Context, op operation.Operation, fldPath *field.Path, items, oldItems []T, keyFields ...[]string) field.ErrorList {
	if op.Type != operation.Update {
		return nil
	}

	// Determine the key function to use
	keyFunc := defaultItemKey[T]
	if len(keyFields) > 0 && len(keyFields[0]) > 0 {
		keyFunc = makeListMapKeyFunc[T](keyFields[0])
	}

	// Check if any new items were added
	oldItemsMap := make(map[string]bool)
	for _, item := range oldItems {
		key := keyFunc(item)
		oldItemsMap[key] = true
	}

	for i, item := range items {
		key := keyFunc(item)
		if !oldItemsMap[key] {
			return field.ErrorList{
				field.Forbidden(fldPath.Index(i), "cannot add new items"),
			}
		}
	}

	return nil
}

// NoRemoveItem verifies that items cannot be removed from a list.
// The optional keyFields parameter specifies the fields to use for list map keys.
func NoRemoveItem[T any](_ context.Context, op operation.Operation, fldPath *field.Path, items, oldItems []T, keyFields ...[]string) field.ErrorList {
	if op.Type != operation.Update {
		return nil
	}

	// Determine the key function to use
	keyFunc := defaultItemKey[T]
	if len(keyFields) > 0 && len(keyFields[0]) > 0 {
		keyFunc = makeListMapKeyFunc[T](keyFields[0])
	}

	// Check if any items were removed
	itemsMap := make(map[string]bool)
	for _, item := range items {
		key := keyFunc(item)
		itemsMap[key] = true
	}

	for _, oldItem := range oldItems {
		key := keyFunc(oldItem)
		if !itemsMap[key] {
			return field.ErrorList{
				field.Forbidden(fldPath, "cannot remove items"),
			}
		}
	}

	return nil
}

// NoRemoveLastItem verifies that at least one item must remain in a list (for required lists).
// The optional keyFields parameter specifies the fields to use for list map keys.
func NoRemoveLastItem[T any](_ context.Context, op operation.Operation, fldPath *field.Path, items, oldItems []T, keyFields ...[]string) field.ErrorList {
	if op.Type != operation.Update {
		return nil
	}

	// If the new list is empty but the old list had items, that's removing the last item
	if len(items) == 0 && len(oldItems) > 0 {
		return field.ErrorList{
			field.Required(fldPath, "cannot remove all items from required list"),
		}
	}

	// Otherwise use the regular NoRemoveItem logic
	return NoRemoveItem[T](context.Background(), op, fldPath, items, oldItems, keyFields...)
}

// NoAddItemMap verifies that entries cannot be added to a map.
func NoAddItemMap[K comparable, V any](_ context.Context, op operation.Operation, fldPath *field.Path, m, oldM map[K]V) field.ErrorList {
	if op.Type != operation.Update {
		return nil
	}

	for k := range m {
		if _, exists := oldM[k]; !exists {
			return field.ErrorList{
				field.Forbidden(fldPath.Key(fmt.Sprintf("%v", k)), "cannot add new entries"),
			}
		}
	}

	return nil
}

// NoRemoveItemMap verifies that entries cannot be removed from a map.
func NoRemoveItemMap[K comparable, V any](_ context.Context, op operation.Operation, fldPath *field.Path, m, oldM map[K]V) field.ErrorList {
	if op.Type != operation.Update {
		return nil
	}

	// To ensure deterministic behavior, we'll collect all missing keys and sort them
	var missingKeys []string
	for k := range oldM {
		if _, exists := m[k]; !exists {
			missingKeys = append(missingKeys, fmt.Sprintf("%v", k))
		}
	}

	if len(missingKeys) == 0 {
		return nil
	}

	// Sort the keys to ensure deterministic error messages
	sort.Strings(missingKeys)

	// Return error for the first missing key (sorted order)
	return field.ErrorList{
		field.Forbidden(fldPath.Key(missingKeys[0]), "cannot remove entries"),
	}
}

// isUnsetValue determines if a value is "unset" for any type T.
// This follows the thread design where struct fields are never "unset".
// For strings, we consider empty string as "unset" for the purpose of
// these validators, even though the thread design says otherwise, because
// that's how the existing Kubernetes APIs work (e.g., PVC.volumeName).
func isUnsetValue[T any](value T) bool {
	return isUnset(value)
}

// isUnset is the helper function that determines if a value is "unset"
func isUnset(value interface{}) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}

	// Handle pointers - including **T cases
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return true
		}
		v = v.Elem()
	}

	// Now check the actual value
	switch v.Kind() {
	case reflect.String:
		// For the update validators, we need to treat empty string as "unset"
		// to match the existing Kubernetes API behavior (e.g., PVC.volumeName)
		// This differs from the thread design but matches current implementation
		return v.String() == ""
	case reflect.Slice, reflect.Map:
		// nil or empty collections are considered unset
		return v.IsNil() || v.Len() == 0
	case reflect.Struct:
		// Per thread design: "Struct-type fields always have a value, and cannot be set or unset"
		return false
	case reflect.Interface:
		return v.IsNil()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.Bool:
		// For primitive types, we cannot distinguish between "unset" and "zero value"
		// This is a limitation of Go's type system. In practice, these should be
		// pointer types if "unset" needs to be distinguished from zero.
		return false
	default:
		// For other types, structs are never "unset"
		return false
	}
}

// defaultItemKey generates a string key for an item using fmt.Sprintf
func defaultItemKey[T any](item T) string {
	return fmt.Sprintf("%v", item)
}

// makeListMapKeyFunc creates a function that generates keys based on specified fields
func makeListMapKeyFunc[T any](keyFields []string) func(T) string {
	return func(item T) string {
		v := reflect.ValueOf(item)

		// Handle pointer types
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return ""
			}
			v = v.Elem()
		}

		// Must be a struct
		if v.Kind() != reflect.Struct {
			// Fall back to default behavior
			return defaultItemKey(item)
		}

		// Build the key from the specified fields
		var keyParts []string
		for _, fieldName := range keyFields {
			field := v.FieldByName(fieldName)
			if !field.IsValid() {
				// Field doesn't exist, fall back to default
				return defaultItemKey(item)
			}
			keyParts = append(keyParts, fmt.Sprintf("%v", field.Interface()))
		}

		// Join all key parts
		return strings.Join(keyParts, "|")
	}
}
