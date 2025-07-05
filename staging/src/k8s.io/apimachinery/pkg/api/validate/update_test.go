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
	"testing"

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

func TestNoSet(t *testing.T) {
	ctx := context.Background()
	path := field.NewPath("test")

	tests := []struct {
		name      string
		value     *string
		oldValue  *string
		expectErr bool
	}{
		{
			name:     "both nil",
			value:    nil,
			oldValue: nil,
		},
		{
			name:     "both set same",
			value:    ptr.To("test"),
			oldValue: ptr.To("test"),
		},
		{
			name:     "both set different",
			value:    ptr.To("new"),
			oldValue: ptr.To("old"),
		},
		{
			name:      "unset to set",
			value:     ptr.To("test"),
			oldValue:  nil,
			expectErr: true,
		},
		{
			name:     "set to unset",
			value:    nil,
			oldValue: ptr.To("test"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := NoSet(ctx, operation.Operation{Type: operation.Update}, path, tt.value, tt.oldValue)
			if tt.expectErr && len(errs) == 0 {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && len(errs) > 0 {
				t.Errorf("unexpected error: %v", errs)
			}
		})
	}
}

func TestNoUnset(t *testing.T) {
	ctx := context.Background()
	path := field.NewPath("test")

	tests := []struct {
		name      string
		value     *string
		oldValue  *string
		expectErr bool
	}{
		{
			name:     "both nil",
			value:    nil,
			oldValue: nil,
		},
		{
			name:     "both set same",
			value:    ptr.To("test"),
			oldValue: ptr.To("test"),
		},
		{
			name:     "both set different",
			value:    ptr.To("new"),
			oldValue: ptr.To("old"),
		},
		{
			name:     "unset to set",
			value:    ptr.To("test"),
			oldValue: nil,
		},
		{
			name:      "set to unset",
			value:     nil,
			oldValue:  ptr.To("test"),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := NoUnset(ctx, operation.Operation{Type: operation.Update}, path, tt.value, tt.oldValue)
			if tt.expectErr && len(errs) == 0 {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && len(errs) > 0 {
				t.Errorf("unexpected error: %v", errs)
			}
		})
	}
}

func TestNoModify(t *testing.T) {
	ctx := context.Background()
	path := field.NewPath("test")

	tests := []struct {
		name      string
		value     string
		oldValue  string
		expectErr bool
	}{
		{
			name:     "both empty",
			value:    "",
			oldValue: "",
		},
		{
			name:     "same value",
			value:    "test",
			oldValue: "test",
		},
		{
			name:      "different values",
			value:     "new",
			oldValue:  "old",
			expectErr: true,
		},
		{
			name:     "unset to set",
			value:    "test",
			oldValue: "",
		},
		{
			name:     "set to unset",
			value:    "",
			oldValue: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := NoModify(ctx, operation.Operation{Type: operation.Update}, path, tt.value, tt.oldValue)
			if tt.expectErr && len(errs) == 0 {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && len(errs) > 0 {
				t.Errorf("unexpected error: %v", errs)
			}
		})
	}
}

func TestNoAddItem(t *testing.T) {
	ctx := context.Background()
	path := field.NewPath("test")

	tests := []struct {
		name      string
		items     []string
		oldItems  []string
		expectErr bool
	}{
		{
			name:     "both empty",
			items:    []string{},
			oldItems: []string{},
		},
		{
			name:     "same items",
			items:    []string{"a", "b"},
			oldItems: []string{"a", "b"},
		},
		{
			name:     "reordered items",
			items:    []string{"b", "a"},
			oldItems: []string{"a", "b"},
		},
		{
			name:      "added item",
			items:     []string{"a", "b", "c"},
			oldItems:  []string{"a", "b"},
			expectErr: true,
		},
		{
			name:     "removed item",
			items:    []string{"a"},
			oldItems: []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := NoAddItem(ctx, operation.Operation{Type: operation.Update}, path, tt.items, tt.oldItems)
			if tt.expectErr && len(errs) == 0 {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && len(errs) > 0 {
				t.Errorf("unexpected error: %v", errs)
			}
		})
	}
}

func TestNoRemoveItem(t *testing.T) {
	ctx := context.Background()
	path := field.NewPath("test")

	tests := []struct {
		name      string
		items     []string
		oldItems  []string
		expectErr bool
	}{
		{
			name:     "both empty",
			items:    []string{},
			oldItems: []string{},
		},
		{
			name:     "same items",
			items:    []string{"a", "b"},
			oldItems: []string{"a", "b"},
		},
		{
			name:     "reordered items",
			items:    []string{"b", "a"},
			oldItems: []string{"a", "b"},
		},
		{
			name:     "added item",
			items:    []string{"a", "b", "c"},
			oldItems: []string{"a", "b"},
		},
		{
			name:      "removed item",
			items:     []string{"a"},
			oldItems:  []string{"a", "b"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := NoRemoveItem(ctx, operation.Operation{Type: operation.Update}, path, tt.items, tt.oldItems)
			if tt.expectErr && len(errs) == 0 {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && len(errs) > 0 {
				t.Errorf("unexpected error: %v", errs)
			}
		})
	}
}

func TestNoUnsetDirectly(t *testing.T) {
	ctx := context.Background()
	path := field.NewPath("test")

	// Test with *string
	oldStr := "value"
	newStr := ""

	errs := NoUnset(ctx, operation.Operation{Type: operation.Update}, path, &newStr, &oldStr)
	if len(errs) != 1 {
		t.Errorf("Expected 1 error, got %d: %v", len(errs), errs)
	}
}

func TestNoUnsetLikeGenerated(t *testing.T) {
	ctx := context.Background()
	op := operation.Operation{Type: operation.Update}

	// Create values like the generated code would
	oldValue := "initial-value"
	newValue := "" // cleared

	// Call it like the generated code does
	var errs field.ErrorList
	errs = append(errs,
		func(fldPath *field.Path, obj, oldObj *string) (errs field.ErrorList) {
			// This is the early return check from generated code
			if op.Type == operation.Update && (obj == oldObj || (obj != nil && oldObj != nil && *obj == *oldObj)) {
				t.Logf("Early return check: obj=%p oldObj=%p", obj, oldObj)
				t.Logf("  obj=%v, oldObj=%v", obj, oldObj)
				t.Logf("  *obj=%q, *oldObj=%q", *obj, *oldObj)
				t.Logf("  Would return early: %v", *obj == *oldObj)
				// Don't actually return, continue to test
			}

			// Test what OptionalValue might do
			t.Logf("About to call validators")

			// Call NoModify
			modifyErrs := NoModify(ctx, op, fldPath, obj, oldObj)
			t.Logf("NoModify returned %d errors: %v", len(modifyErrs), modifyErrs)
			errs = append(errs, modifyErrs...)

			// Call NoUnset
			unsetErrs := NoUnset(ctx, op, fldPath, obj, oldObj)
			t.Logf("NoUnset returned %d errors: %v", len(unsetErrs), unsetErrs)
			errs = append(errs, unsetErrs...)

			return
		}(field.NewPath("setOnceField"), &newValue, &oldValue)...)

	t.Logf("Total errors: %d", len(errs))
	if len(errs) == 0 {
		t.Errorf("Expected errors but got none")
	} else {
		t.Logf("Got expected errors: %v", errs)
	}
}

func TestOptionalValueBehavior(t *testing.T) {
	ctx := context.Background()
	op := operation.Operation{Type: operation.Update}

	// Test with empty string
	oldValue := "initial-value"
	newValue := ""

	// See what OptionalValue returns for this case
	errs := OptionalValue(ctx, op, field.NewPath("test"), &newValue, &oldValue)
	t.Logf("OptionalValue returned %d errors: %v", len(errs), errs)

	if len(errs) > 0 {
		t.Logf("OptionalValue is preventing other validators from running!")
	}
}
