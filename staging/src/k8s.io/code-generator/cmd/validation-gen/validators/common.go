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

package validators

import (
	"k8s.io/gengo/v2/parser/tags"
	"k8s.io/gengo/v2/types"
)

const (
	// libValidationPkg is the pkgpath to our "standard library" of validation
	// functions.
	libValidationPkg = "k8s.io/apimachinery/pkg/api/validate"
)

func getMemberByJSON(t *types.Type, jsonName string) *types.Member {
	for i := range t.Members {
		if jsonTag, ok := tags.LookupJSON(t.Members[i]); ok {
			if jsonTag.Name == jsonName {
				return &t.Members[i]
			}
		}
	}
	return nil
}

// isNilableType returns true if the argument type can be compared to nil.
func isNilableType(t *types.Type) bool {
	for t.Kind == types.Alias {
		t = t.Underlying
	}
	switch t.Kind {
	case types.Pointer, types.Map, types.Slice, types.Interface: // Note: Arrays are not nilable
		return true
	}
	return false
}

// NativeType returns the Go native type of the argument type, with any
// intermediate typedefs removed. Go itself already flattens typedefs, but this
// handles it in the unlikely event that we ever fix that.
//
// Examples:
// * Trivial:
//   - given `int`, returns `int`
//   - given `*int`, returns `*int`
//   - given `[]int`, returns `[]int`
//
// * Typedefs
//   - given `type X int; X`, returns `int`
//   - given `type X int; []X`, returns `[]X`
//
// * Typedefs and pointers:
//   - given `type X int; *X`, returns `*int`
//   - given `type X *int; *X`, returns `**int`
//   - given `type X []int; X`, returns `[]int`
//   - given `type X []int; *X`, returns `*[]int`
func NativeType(t *types.Type) *types.Type {
	ptrs := 0
	for {
		if t.Kind == types.Alias {
			t = t.Underlying
		} else if t.Kind == types.Pointer {
			ptrs++
			t = t.Elem
		} else {
			break
		}
	}
	for range ptrs {
		t = types.PointerTo(t)
	}
	return t
}

// NonPointer returns the value-type of a possibly pointer type. If type is not
// a pointer, it returns the input type.
func NonPointer(t *types.Type) *types.Type {
	for t.Kind == types.Pointer {
		t = t.Elem
	}
	return t
}

// rootTypeString returns a string representation of the relationship between
// src and dst types, for use in error messages.
func rootTypeString(src, dst *types.Type) string {
	if src == dst {
		return src.String()
	}
	return src.String() + " -> " + dst.String()
}

// IsDirectComparable returns true if the type is safe to compare using "==".
// It is similar to gengo.IsComparable, but is doesn't consider Pointers
// as comparable.
// This would be used for validation ratcheting to check whether this type can directly be compared.
func IsDirectComparable(t *types.Type) bool {
	switch t.Kind {
	case types.Builtin:
		return true
	case types.Struct:
		for _, f := range t.Members {
			if !IsDirectComparable(f.Type) {
				return false
			}
		}
		return true
	case types.Array:
		return IsDirectComparable(t.Elem)
	case types.Alias:
		return IsDirectComparable(t.Underlying)
	}
	return false
}

// GetMapValueType returns the value type of a map, handling nested types like
// map[string][]T by returning T for validation purposes.
func GetMapValueType(t *types.Type) *types.Type {
	if t.Kind != types.Map {
		return nil
	}
	valueType := NativeType(t.Elem)
	// If the map value is a slice, return the slice element type
	if valueType.Kind == types.Slice {
		return NativeType(valueType.Elem)
	}
	return valueType
}

// IsMapOfSlice returns true if the type is a map whose values are slices.
func IsMapOfSlice(t *types.Type) bool {
	t = NativeType(t)
	if t.Kind != types.Map {
		return false
	}
	valueType := NativeType(t.Elem)
	return valueType.Kind == types.Slice
}

// GetSliceElementType returns the element type of a slice, handling aliases.
func GetSliceElementType(t *types.Type) *types.Type {
	t = NativeType(t)
	if t.Kind != types.Slice && t.Kind != types.Array {
		return nil
	}
	return NativeType(t.Elem)
}

// IsTypeSupported checks if a type is supported by the validation generator.
// This is used to validate complex nested types.
func IsTypeSupported(t *types.Type) bool {
	t = NativeType(t)
	switch t.Kind {
	case types.Builtin, types.Struct, types.Interface:
		return true
	case types.Pointer:
		// Check what it points to
		pointee := NativeType(t.Elem)
		switch pointee.Kind {
		case types.Pointer, types.Slice, types.Array, types.Map:
			return false // No pointers to pointers, slices, arrays, or maps
		}
		return IsTypeSupported(pointee)
	case types.Slice, types.Array:
		elem := NativeType(t.Elem)
		switch elem.Kind {
		case types.Pointer:
			return false // No lists of pointers
		case types.Slice:
			// Only [][]byte is allowed
			if elem.Kind == types.Slice && NativeType(elem.Elem) == types.Byte {
				return true
			}
			return false
		case types.Map:
			return false // No lists of maps
		}
		return IsTypeSupported(elem)
	case types.Map:
		// Keys must be strings
		if NativeType(t.Key) != types.String {
			return false
		}
		// Check value type
		elem := NativeType(t.Elem)
		switch elem.Kind {
		case types.Pointer:
			return false // No maps of pointers
		case types.Map:
			return false // No maps of maps
		case types.Slice:
			// Maps of slices are now supported!
			// Check the slice element type
			sliceElem := NativeType(elem.Elem)
			switch sliceElem.Kind {
			case types.Pointer:
				return false // No maps of lists of pointers
			case types.Slice:
				// Only maps of [][]byte are allowed
				if sliceElem.Kind == types.Slice && NativeType(sliceElem.Elem) == types.Byte {
					return true
				}
				return false
			case types.Map:
				return false // No maps of lists of maps
			}
			return IsTypeSupported(sliceElem)
		}
		return IsTypeSupported(elem)
	case types.Alias:
		return IsTypeSupported(t.Underlying)
	}
	return false
}

// GetNestedValueType returns the deepest value type for nested collections.
// For example, for map[string][]T, it returns T.
func GetNestedValueType(t *types.Type) *types.Type {
	t = NativeType(t)
	switch t.Kind {
	case types.Map:
		return GetNestedValueType(t.Elem)
	case types.Slice, types.Array:
		return GetNestedValueType(t.Elem)
	default:
		return t
	}
}
