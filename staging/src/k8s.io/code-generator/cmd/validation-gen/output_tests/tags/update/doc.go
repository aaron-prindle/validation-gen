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

// +k8s:validation-gen=TypeMeta
// +k8s:validation-gen-scheme-registry=k8s.io/code-generator/cmd/validation-gen/testscheme.Scheme

// This is a test package for the +k8s:update tag.
package update

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

type UpdateTestStruct struct {
	TypeMeta int

	// Field that can be set once after creation (like PVC.volumeName)
	// This is the "immutable once set" pattern from the design doc
	// +k8s:update=`NoModify,NoUnset`
	SetOnceField string `json:"setOnceField"`

	// Field that must be set at creation or remain unset forever
	// If unset at creation, it can never be set
	// +k8s:update=`NoSet`
	CreateOnlyField string `json:"createOnlyField"`

	// Append-only list
	// +k8s:update=`NoRemoveItem,NoUnset`
	AppendOnlyList []string `json:"appendOnlyList"`

	// Immutable list (no adds or removes)
	// +k8s:update=`NoAddItem,NoRemoveItem`
	ImmutableList []Item `json:"immutableList"`

	// Map where entries cannot be removed
	// +k8s:update=`NoRemoveItem`
	NoRemoveMap map[string]string `json:"noRemoveMap"`

	// Field that cannot be cleared once set (requiredOnceSet pattern)
	// Note: This is a pointer field, so it needs special handling
	// +k8s:update=`NoUnset`
	RequiredOnceSetField *string `json:"requiredOnceSetField"`

	// Truly immutable field - whatever value it has at creation
	// (set or unset) cannot be changed
	// Note: removing +k8s:optional to avoid OptionalValue issues
	// +k8s:immutable
	TrulyImmutableField string `json:"trulyImmutableField"`

	// Required immutable field - must be set at creation and cannot change
	// +k8s:required
	// +k8s:immutable
	RequiredImmutableField string `json:"requiredImmutableField"`
}

type Item struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
