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
	"strings"

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/api/validate/util"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// GetFieldFunc is a function that extracts a field from a type and returns a
// nilable value.
type GetFieldFunc[Tstruct any, Tfield any] func(*Tstruct) Tfield

// GetMapFunc is a function that returns a map.
type GetMapFunc func() map[string]string

// Subfield validates a subfield of a struct against a validator function.
func Subfield[Tstruct any, Tfield any](ctx context.Context, op operation.Operation, fldPath *field.Path, newStruct, oldStruct *Tstruct,
	fldName string, getField GetFieldFunc[Tstruct, Tfield], validator ValidateFunc[Tfield]) field.ErrorList {
	var errs field.ErrorList
	newVal := getField(newStruct)
	var oldVal Tfield
	if oldStruct != nil {
		oldVal = getField(oldStruct)
	}
	errs = append(errs, validator(ctx, op, fldPath.Child(fldName), newVal, oldVal)...)
	return errs
}

func ListMapElementByKey[TList ~[]TItem, TItem any](
	ctx context.Context, op operation.Operation, fldPath *field.Path,
	newList, oldList TList, getMap GetMapFunc,
	elementValidator func(ctx context.Context, op operation.Operation, fldPath *field.Path, newObj, oldObj *TItem) field.ErrorList,
) field.ErrorList {

	var errs field.ErrorList
	m := getMap()
	elementPathForSelector := fldPath.Key(util.GeneratePathForMap(m))
	processedOldIndices := make(map[int]bool)

	for i := range newList {
		curI := newList[i]
		// curI must match all selectors - ex: {"type":"Approved", "status":"True"}.
		if !matchesAllSelectors(curI, m) {
			continue
		}

		var oldJ *TItem
		for j := range oldList {
			if processedOldIndices[j] {
				continue
			}
			if matchesAllSelectors(oldList[j], m) {
				oldJ = &oldList[j]
				processedOldIndices[j] = true
				break
			}
		}
		// Pass matching elements present in newList and oldList to validator.
		errs = append(errs, elementValidator(ctx, op, elementPathForSelector, &curI, oldJ)...)
	}

	for i := range oldList {
		if !processedOldIndices[i] && matchesAllSelectors(oldList[i], m) {
			// Pass matching elements present only in oldList to validator.
			errs = append(errs, elementValidator(ctx, op, elementPathForSelector, nil, &oldList[i])...)
		}
	}

	return errs
}

func matchesAllSelectors[TItem any](item TItem, m map[string]string) bool {
	for matchK, matchV := range m {
		if fieldV, ok := getReflectedJSONFieldValueAsString(reflect.ValueOf(item), matchK); !ok || fieldV != matchV {
			return false
		}
	}
	return true
}

func getReflectedJSONFieldValueAsString(sVal reflect.Value, jsonKeyName string) (string, bool) {
	// sVal assumed to be a non-pointer struct.
	for i := range sVal.Type().NumField() {
		if jsonName, ok := getJSONFieldName(sVal.Type().Field(i)); !ok || jsonName != jsonKeyName {
			continue
		}

		fieldValue := sVal.Field(i)
		if !fieldValue.CanInterface() {
			return "", false
		}
		// TODO(aaron-prindle) currently this only supports string, string pointer, and string alias types.
		if fieldValue.Kind() == reflect.String {
			return fieldValue.String(), true
		}
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				return "", false
			}
			if fieldValue.Elem().Kind() == reflect.String {
				return fieldValue.Elem().String(), true
			}
		}
		// TODO(aaron-prindle) stripping aliasing more similar to current patterns than this way.
		if fieldValue.Type().ConvertibleTo(reflect.TypeOf("")) {
			return fieldValue.Convert(reflect.TypeOf("")).String(), true
		}
		return "", false
	}
	return "", false
}

func getJSONFieldName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		// Indicates field is ignored.
		return "", false
	}
	if tag == "" {
		// Indicates no json tag.
		return field.Name, true
	}
	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "" {
		// Default name behavior for json field.
		return field.Name, true
	}
	return name, true
}
