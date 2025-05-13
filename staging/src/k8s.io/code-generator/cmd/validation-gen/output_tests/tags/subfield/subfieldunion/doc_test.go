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

package subfieldunion

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestSubfieldWithUnionMemberPayload(t *testing.T) {
	st := localSchemeBuilder.Test(t)

	errorMsgMustSpecifyExactlyOne := "must specify exactly one of: `_subfield_Items_status_is_True_and_type_is_Approved`, `_subfield_Items_status_is_True_and_type_is_Denied`"

	st.Value(&Struct{
		Items: []Item{},
	}).ExpectInvalid(
		field.Invalid(nil, "", errorMsgMustSpecifyExactlyOne),
	)

	st.Value(&Struct{
		Items: []Item{
			{Type: "Approved", Status: "True"},
		},
	}).ExpectValid()

	st.Value(&Struct{
		Items: []Item{
			{Type: "Denied", Status: "True"},
		},
	}).ExpectValid()

	st.Value(&Struct{
		Items: []Item{
			{Type: "Approved", Status: "True"},
			{Type: "Denied", Status: "True"},
		},
	}).ExpectInvalid(
		field.Invalid(nil, "{_subfield_Items_status_is_True_and_type_is_Approved, _subfield_Items_status_is_True_and_type_is_Denied}", errorMsgMustSpecifyExactlyOne),
	)

	st.Value(&Struct{
		Items: []Item{
			{Type: "Pending", Status: "True"},
			{Type: "Approved", Status: "False"},
		},
	}).ExpectInvalid(
		field.Invalid(nil, "", errorMsgMustSpecifyExactlyOne),
	)

	st.Value(&Struct{
		Items: []Item{
			{Type: "Approved", Status: "True"},
			{Type: "Approved", Status: "True"},
		},
	}).ExpectValid()

	st.Value(&Struct{
		Items: nil,
	}).ExpectInvalid(
		field.Invalid(nil, "", errorMsgMustSpecifyExactlyOne),
	)
}
