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

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// GetFieldFunc is a function that extracts a field from a type and returns a
// nilable value.
type GetFieldFunc[Tstruct any, Tfield any] func(*Tstruct) Tfield

// MatchFn takes a pointer to an item and returns true if it matches the criteria.
type MatchFn[T any] func(*T) bool

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
	newList, oldList TList,
	matches MatchFn[TItem],
	elementValidator func(ctx context.Context, op operation.Operation, fldPath *field.Path, newObj, oldObj *TItem) field.ErrorList,
) field.ErrorList {
	var errs field.ErrorList
	processedOldIndices := make(map[int]bool)

	for i := range newList {
		curI := &newList[i]
		if !matches(curI) {
			continue
		}
		var oldJ *TItem

		for j := range oldList {
			if processedOldIndices[j] {
				continue
			}

			if matches(&oldList[j]) {
				oldJ = &oldList[j]
				processedOldIndices[j] = true
				break
			}
		}
		// Pass matching element present in newList and oldList to validator.
		errs = append(errs, elementValidator(ctx, op, fldPath.Index(i), curI, oldJ)...)
	}

	for i := range oldList {
		oldI := &oldList[i]
		if !processedOldIndices[i] && matches(oldI) {
			// Pass matching element present only in oldList to validator.
			errs = append(errs, elementValidator(ctx, op, fldPath.Index(i), nil, oldI)...)
		}
	}
	return errs
}
