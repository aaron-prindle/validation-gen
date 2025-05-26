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

// +k8s:validation-gen=TypeMeta
// +k8s:validation-gen-scheme-registry=k8s.io/code-generator/cmd/validation-gen/testscheme.Scheme

// This is a test package for testing +k8s:eachKey on maps of slices.
package mapofsliceskey

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

// +k8s:validateFalse="type TestStruct"
type TestStruct struct {
	TypeMeta int

	// Basic key validation on map of string slices
	// +k8s:eachKey=+k8s:validateFalse="key BasicKeys"
	BasicKeys map[string][]string `json:"basicKeys"`

	// Multiple key validations
	// +k8s:eachKey=+k8s:validateFalse="key MultipleKeys #1"
	// +k8s:eachKey=+k8s:validateFalse="key MultipleKeys #2"
	MultipleKeys map[string][]string `json:"multipleKeys"`

	// Key validation combined with value validation
	// +k8s:eachKey=+k8s:validateFalse="key KeysAndSlices"
	// +k8s:eachVal=+k8s:validateFalse="slice KeysAndSlices"
	KeysAndSlices map[string][]string `json:"keysAndSlices"`

	// Key validation with nested value validation
	// +k8s:eachKey=+k8s:validateFalse="key KeysAndElements"
	// +k8s:eachVal=+k8s:eachVal=+k8s:validateFalse="element KeysAndElements"
	KeysAndElements map[string][]string `json:"keysAndElements"`

	// Key validation on map of integer slices
	// +k8s:eachKey=+k8s:validateFalse="key IntSliceKeys"
	// +k8s:eachVal=+k8s:eachVal=+k8s:validateFalse="element IntSliceKeys"
	IntSliceKeys map[string][]int `json:"intSliceKeys"`

	// Key validation on map of struct slices
	// +k8s:eachKey=+k8s:validateFalse="key StructSliceKeys"
	// +k8s:eachVal=+k8s:validateFalse="slice StructSliceKeys"
	// +k8s:eachVal=+k8s:eachVal=+k8s:validateFalse="element StructSliceKeys"
	StructSliceKeys map[string][]User `json:"structSliceKeys"`

	// Key validation with typedef values
	// +k8s:eachKey=+k8s:validateFalse="key TypedefKeys"
	// +k8s:validateFalse="field TestStruct.TypedefKeys"
	TypedefKeys map[string]ValueList `json:"typedefKeys"`

	// No validation - control case
	UnvalidatedMap map[string][]string `json:"unvalidatedMap"`
}

// User is a simple user struct
// +k8s:validateFalse="type User"
type User struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// ValueList is a typedef for a string slice
// +k8s:validateFalse="type ValueList"
type ValueList []string
