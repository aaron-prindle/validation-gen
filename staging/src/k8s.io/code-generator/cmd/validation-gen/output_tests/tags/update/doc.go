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

	// Primitive field tests (non-pointer)

	// Can't be set after creation
	// +k8s:optional
	// +k8s:update=`NoSet`
	PrimitiveNoSet string `json:"primitiveNoSet"`

	// Can't be unset once set
	// +k8s:optional
	// +k8s:update=`NoUnset`
	PrimitiveNoUnset string `json:"primitiveNoUnset"`

	// Can't be modified once set (but can be set/unset)
	// +k8s:optional
	// +k8s:update=`NoModify`
	PrimitiveNoModify string `json:"primitiveNoModify"`

	// Fully restricted primitive (can't set, unset, or modify)
	// +k8s:optional
	// +k8s:update=`NoSet,NoUnset,NoModify`
	PrimitiveFullyRestricted string `json:"primitiveFullyRestricted"`

	// Required field that can't be modified
	// +k8s:required
	// +k8s:update=`NoModify`
	PrimitiveRequiredNoModify string `json:"primitiveRequiredNoModify"`

	// Set-once field (like PVC.volumeName)
	// +k8s:optional
	// +k8s:update=`NoModify,NoUnset`
	PrimitiveSetOnce string `json:"primitiveSetOnce"`

	// Test different primitive types
	// +k8s:optional
	// +k8s:update=`NoModify`
	IntNoModify int `json:"intNoModify"`

	// +k8s:optional
	// +k8s:update=`NoModify`
	BoolNoModify bool `json:"boolNoModify"`

	// +k8s:optional
	// +k8s:update=`NoModify`
	Float64NoModify float64 `json:"float64NoModify"`

	// Optional + Default tests (effectively required)

	// Optional with default - NoModify (can't change from default)
	// +k8s:optional
	// +default="default-value"
	// +k8s:update=`NoModify`
	OptionalDefaultNoModify string `json:"optionalDefaultNoModify"`

	// Optional with default - NoSet (but this is meaningless since field is required)
	// +k8s:optional
	// +default="default-value"
	// +k8s:update=`NoSet`
	OptionalDefaultNoSet string `json:"optionalDefaultNoSet"`

	// Optional with default - NoUnset (can't clear the default)
	// +k8s:optional
	// +default="default-value"
	// +k8s:update=`NoUnset`
	OptionalDefaultNoUnset string `json:"optionalDefaultNoUnset"`

	// Optional int with default - NoModify
	// +k8s:optional
	// +default=42
	// +k8s:update=`NoModify`
	OptionalDefaultIntNoModify int `json:"optionalDefaultIntNoModify"`

	// Optional bool with default - NoModify
	// +k8s:optional
	// +default=true
	// +k8s:update=`NoModify`
	OptionalDefaultBoolNoModify bool `json:"optionalDefaultBoolNoModify"`

	// Struct field tests (non-pointer)

	// Struct fields can only use NoModify (always set, never unset)
	// +k8s:update=`NoModify`
	StructNoModify TestStruct `json:"structNoModify"`

	// Pointer field tests

	// Can't be set after creation
	// +k8s:optional
	// +k8s:update=`NoSet`
	PointerNoSet *string `json:"pointerNoSet"`

	// Can't be unset once set
	// +k8s:optional
	// +k8s:update=`NoUnset`
	PointerNoUnset *string `json:"pointerNoUnset"`

	// Can't be modified once set (but can be set/unset)
	// +k8s:optional
	// +k8s:update=`NoModify`
	PointerNoModify *string `json:"pointerNoModify"`

	// Fully restricted pointer (can't set, unset, or modify)
	// +k8s:optional
	// +k8s:update=`NoSet,NoUnset,NoModify`
	PointerFullyRestricted *string `json:"pointerFullyRestricted"`

	// Optional pointer with default - effectively required
	// +k8s:optional
	// +default="pointer-default"
	// +k8s:update=`NoModify`
	OptionalDefaultPointerNoModify *string `json:"optionalDefaultPointerNoModify"`

	// Lists and maps - no update constraints supported in this PR
	// These are here to ensure we don't accidentally generate validators for them
	// +k8s:optional
	ListField []string `json:"listField"`

	// +k8s:optional
	MapField map[string]string `json:"mapField"`
}

type TestStruct struct {
	StringField    string   `json:"stringField"`
	StringPtrField *string  `json:"stringPtrField"`
	SliceField     []string `json:"sliceField"`
}
