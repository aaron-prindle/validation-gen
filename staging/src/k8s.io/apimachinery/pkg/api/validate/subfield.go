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
	"k8s.io/apimachinery/pkg/api/validate/util"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type GetFieldFunc[Tstruct any, Tfield any] func(*Tstruct) Tfield

// MatchFunc takes a pointer to an item and returns true if it matches the criteria.
type MatchFunc[TItem any] func(item *TItem) bool

// GetSelectorMapForPathFunc returns a map - needed because the selector map is still used for generating the field.Path key.
// TODO(aaron-prindle) ListMapElementByKey could take the map as a separate arg just for pathing.
type GetSelectorMapForPathFunc func() map[string]string

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
	getSelectorMap GetSelectorMapForPathFunc,
	matches MatchFunc[TItem],
	elementValidator func(ctx context.Context, op operation.Operation, fldPath *field.Path, newObj, oldObj *TItem) field.ErrorList,
) field.ErrorList {
	var errs field.ErrorList
	selectorMapForPath := getSelectorMap()
	elementPathForSelector := fldPath.Key(util.GeneratePathForMap(selectorMapForPath))
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
			curOldJPtr := &oldList[j]
			if matches(curOldJPtr) {
				oldJ = curOldJPtr
				processedOldIndices[j] = true
				break
			}
		}
		// Pass matching element present in newList and oldList to validator.
		errs = append(errs, elementValidator(ctx, op, elementPathForSelector, curI, oldJ)...)
	}

	for i := range oldList {
		curOldI := &oldList[i]
		if !processedOldIndices[i] && matches(curOldI) {
			// Pass matching element present only in oldList to validator.
			errs = append(errs, elementValidator(ctx, op, elementPathForSelector, nil, curOldI)...)
		}
	}
	return errs
}
