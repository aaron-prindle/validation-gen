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
	"reflect"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// NoSet verifies that a field cannot be set (transition from unset to set).
func NoSet[T any](_ context.Context, op operation.Operation, fldPath *field.Path, value, oldValue T) field.ErrorList {
	if op.Type != operation.Update {
		return nil
	}

	valueIsUnset := isUnset(value)
	oldValueIsUnset := isUnset(oldValue)

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

	valueIsUnset := isUnset(value)
	oldValueIsUnset := isUnset(oldValue)

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

	valueIsUnset := isUnset(value)
	oldValueIsUnset := isUnset(oldValue)

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

// isUnset determines if a value is "unset" for any type T.
// This follows the same patterns as the forbidden validators:
// - For pointers: nil is unset, or the pointed-to value is unset
// - For slices/maps: nil or empty is unset
// - For primitives: zero value is unset (including empty string)
// - For structs: never unset (always have a value)
func isUnset[T any](value T) bool {
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return true
	}

	// Handle pointers
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return true
		}
		// Dereference the pointer and check if the underlying value is unset
		return isUnset(v.Elem().Interface())
	}

	// Check the value based on its kind
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		// nil or empty collections are considered unset
		return v.IsNil() || v.Len() == 0
	case reflect.Struct:
		// Struct fields always have a value and cannot be unset
		return false
	case reflect.Interface:
		return v.IsNil()
	case reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		// For all other types (including primitives and strings),
		// check if the value equals its zero value
		zero := reflect.Zero(v.Type())
		return reflect.DeepEqual(v.Interface(), zero.Interface())
	}
}
