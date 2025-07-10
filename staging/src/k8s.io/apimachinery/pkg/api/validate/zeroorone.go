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

package validate

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/api/operation"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ZeroOrOneOfUnion verifies that at most one member of a union is specified.
//
// UnionMembership must define all the members of the union.
//
// For example:
//
//	var UnionMembershipForABC := validate.NewUnionMembership([2]string{"a", "A"}, [2]string{"b", "B"}, [2]string{"c", "C"})
//	func ValidateABC(ctx context.Context, op operation.Operation, fldPath *field.Path, in *ABC) (errs fields.ErrorList) {
//		errs = append(errs, ZeroOrOneOfUnion(ctx, op, fldPath, in, oldIn, UnionMembershipForABC,
//			func(in *ABC) bool { return in.A != nil },
//			func(in *ABC) bool { return in.B != ""},
//			func(in *ABC) bool { return in.C != 0 },
//		)...)
//		return errs
//	}
func ZeroOrOneOfUnion[T any](ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj T, union *UnionMembership, isSetFns ...ExtractorFn[T, bool]) field.ErrorList {
	options := UnionValidationOptions{
		ErrorForEmpty: nil,
		ErrorForMultiple: func(fldPath *field.Path, specifiedFields []string, allFields []string) *field.Error {
			return field.Invalid(fldPath, fmt.Sprintf("{%s}", strings.Join(specifiedFields, ", ")),
				fmt.Sprintf("must specify at most one of: %s", strings.Join(allFields, ", ")))
		},
	}

	errs := unionValidate(op, fldPath, obj, oldObj, union, options, isSetFns...)
	return errs
}

// ZeroOrOneOfDiscriminatedUnion verifies specified union member matches the discriminator, allowing empty unions.
//
// UnionMembership must define all the members of the union and the discriminator.
//
// For example:
//
//	var UnionMembershipForABC := validate.NewDiscriminatedUnionMembership("type", [2]string{"a", "A"}, [2]string{"b" "B"}, [2]string{"c", "C"})
//	func ValidateABC(ctx context.Context, op operation.Operation, fldPath, *field.Path, in *ABC) (errs fields.ErrorList) {
//		errs = append(errs, ZeroOrOneOfDiscriminatedUnion(ctx, op, fldPath, in, oldIn, UnionMembershipForABC,
//			func(in *ABC) string { return string(in.Type) },
//			func(in *ABC) bool { return in.A != nil },
//			func(in *ABC) bool { return in.B != ""},
//			func(in *ABC) bool { return in.C != 0 },
//		)...)
//		return errs
//	}
//
// When the discriminator is empty, no fields are required to be set.
// When the discriminator is set, at most one field matching that discriminator value may be set.
func ZeroOrOneOfDiscriminatedUnion[T any, D ~string](ctx context.Context, op operation.Operation, fldPath *field.Path, obj, oldObj T, union *UnionMembership, discriminatorExtractor ExtractorFn[T, D], isSetFns ...ExtractorFn[T, bool]) (errs field.ErrorList) {
	options := UnionValidationOptions{
		ErrorForMultiple: func(fldPath *field.Path, specifiedFields []string, allFields []string) *field.Error {
			return field.Invalid(fldPath, fmt.Sprintf("{%s}", strings.Join(specifiedFields, ", ")),
				fmt.Sprintf("must specify at most one of: %s", strings.Join(allFields, ", ")))
		},
		ErrorForMissingRequired: nil,
	}

	// Special handling for empty discriminator
	discriminatorValue := discriminatorExtractor(obj)
	if string(discriminatorValue) == "" {
		return ZeroOrOneOfUnion(ctx, op, fldPath, obj, oldObj, union, isSetFns...)
	}

	return discriminatedUnionValidate(op, fldPath, obj, oldObj, union, options, discriminatorExtractor, isSetFns...)
}
