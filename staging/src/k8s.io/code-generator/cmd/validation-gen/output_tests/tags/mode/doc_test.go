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

package mode

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
)

func TestStrictUnion(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// Mode A: FieldA required, FieldB forbidden
	st.Value(&StrictUnion{Mode: "A", FieldA: ptr.To("val")}).ExpectValid()
	st.Value(&StrictUnion{Mode: "A"}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Required(field.NewPath("fieldA"), ""),
	})
	st.Value(&StrictUnion{Mode: "A", FieldA: ptr.To("val"), FieldB: ptr.To("val")}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Forbidden(field.NewPath("fieldB"), ""),
	})

	// Mode B: FieldA forbidden, FieldB required
	st.Value(&StrictUnion{Mode: "B", FieldB: ptr.To("val")}).ExpectValid()
	st.Value(&StrictUnion{Mode: "B"}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Required(field.NewPath("fieldB"), ""),
	})
	st.Value(&StrictUnion{Mode: "B", FieldA: ptr.To("val"), FieldB: ptr.To("val")}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Forbidden(field.NewPath("fieldA"), ""),
	})
}

func TestSharedField(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// Valid (optional) in A and B
	st.Value(&SharedField{Mode: "A"}).ExpectValid()
	st.Value(&SharedField{Mode: "A", Shared: ptr.To("val")}).ExpectValid()
	st.Value(&SharedField{Mode: "B"}).ExpectValid()
	st.Value(&SharedField{Mode: "B", Shared: ptr.To("val")}).ExpectValid()

	// Forbidden in C
	st.Value(&SharedField{Mode: "C", Shared: ptr.To("val")}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Forbidden(field.NewPath("shared"), ""),
	})
}

func TestChainedValidation(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// Mode A: Required AND maxLength 5
	st.Value(&ChainedValidation{Mode: "A", Field: ptr.To("abc")}).ExpectValid()
	st.Value(&ChainedValidation{Mode: "A"}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Required(field.NewPath("field"), ""),
	})
	st.Value(&ChainedValidation{Mode: "A", Field: ptr.To("too-long")}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.TooLong(field.NewPath("field"), "too-long", 5),
	})
}

func TestImplicitForbidden(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// Mode A: Optional
	st.Value(&ImplicitForbidden{Mode: "A"}).ExpectValid()
	st.Value(&ImplicitForbidden{Mode: "A", Field: ptr.To("val")}).ExpectValid()

	// Mode B: Not listed, so implicitly Forbidden
	st.Value(&ImplicitForbidden{Mode: "B", Field: ptr.To("val")}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Forbidden(field.NewPath("field"), ""),
	})
}

func TestNonStringDiscriminator(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	// Bool mode
	st.Value(&NonStringDiscriminator{BoolMode: true, BoolField: ptr.To("val")}).ExpectValid()
	st.Value(&NonStringDiscriminator{BoolMode: true}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Required(field.NewPath("boolField"), ""),
	})
	st.Value(&NonStringDiscriminator{BoolMode: false, BoolField: ptr.To("val")}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Forbidden(field.NewPath("boolField"), ""),
	})

	// Int mode
	st.Value(&NonStringDiscriminator{IntMode: 1, IntField: ptr.To("val")}).ExpectValid()
	st.Value(&NonStringDiscriminator{IntMode: 1}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Required(field.NewPath("intField"), ""),
	})
	st.Value(&NonStringDiscriminator{IntMode: 2, IntField: ptr.To("val")}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Forbidden(field.NewPath("intField"), ""),
	})
}

func TestMultipleDiscriminators(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	st.Value(&MultipleDiscriminators{
		NetMode:     "IPv6",
		StorageMode: "S3",
		IPv6Config:  ptr.To("config"),
		S3Bucket:    ptr.To("bucket"),
	}).ExpectValid()

	st.Value(&MultipleDiscriminators{
		NetMode:     "IPv6",
		StorageMode: "S3",
	}).ExpectMatches(field.ErrorMatcher{}.ByType().ByField(), field.ErrorList{
		field.Required(field.NewPath("ipv6Config"), ""),
		field.Required(field.NewPath("s3Bucket"), ""),
	})
}
