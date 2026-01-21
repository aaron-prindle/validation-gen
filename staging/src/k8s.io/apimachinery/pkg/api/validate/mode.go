/*
Copyright The Kubernetes Authors.

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

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ModalRule defines a validation to apply for a specific mode value.
type ModalRule[T any] struct {
	Value      string
	Validation ValidateFunc[T]
}

// Modal validates a value based on a discriminator value.
// It iterates through the rules and applies the first one that matches the discriminator.
// If no rule matches, it applies the defaultValidation if provided.
// The obj and oldObj arguments are the parent objects, which are ignored by Modal
// but are passed by validation-gen.
// The fldPath argument is the path to the field being validated.
func Modal[T any, D any, P any](ctx context.Context, op operation.Operation, _ *field.Path, obj, oldObj P, fldPath *field.Path, value, oldValue T, discriminator D, defaultValidation ValidateFunc[T], rules []ModalRule[T]) field.ErrorList {
	dStr := fmt.Sprintf("%v", discriminator)
	for _, rule := range rules {
		if rule.Value == dStr {
			if rule.Validation == nil {
				return nil
			}
			return rule.Validation(ctx, op, fldPath, value, oldValue)
		}
	}
	if defaultValidation != nil {
		return defaultValidation(ctx, op, fldPath, value, oldValue)
	}
	return nil
}