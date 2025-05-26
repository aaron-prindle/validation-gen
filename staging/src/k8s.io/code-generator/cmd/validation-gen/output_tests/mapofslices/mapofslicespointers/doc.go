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

// This is a test package for maps of slices with allowed pointer types.
package mapofslicespointers

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

// SimpleStruct is a basic struct type
type SimpleStruct struct {
	// +k8s:maxLength=20
	Name string `json:"name"`
	// +k8s:minimum=0
	Value int `json:"value"`
}

// TestStruct tests maps of slices with structs (not pointers)
type TestStruct struct {
	TypeMeta int

	// Maps of slices of structs are allowed (structs contain the pointers, not slices)
	StructSlices map[string][]SimpleStruct `json:"structSlices"`

	// Test with struct containing pointer fields
	ComplexStructSlices map[string][]ComplexStruct `json:"complexStructSlices"`

	// Direct primitive slices (no pointers involved)
	StringSlices map[string][]string `json:"stringSlices"`
}

// ComplexStruct has pointer fields (allowed in structs)
type ComplexStruct struct {
	// Pointer fields in structs are allowed
	OptionalName *string `json:"optionalName,omitempty"`
	OptionalInt  *int    `json:"optionalInt,omitempty"`

	// Nested struct pointer
	Nested *SimpleStruct `json:"nested,omitempty"`

	// Regular fields
	Required string `json:"required"`
}

// Note: The following would NOT be allowed (commented out to prevent compilation errors):
//
// // Maps of slices of pointers are NOT allowed
// type InvalidStruct1 struct {
//     TypeMeta int
//     // This would fail: maps of lists of pointers are not supported
//     PtrSlices map[string][]*SimpleStruct `json:"ptrSlices"`
// }
//
// // Maps of pointers are NOT allowed
// type InvalidStruct2 struct {
//     TypeMeta int
//     // This would fail: maps of pointers are not supported
//     PtrMap map[string]*SimpleStruct `json:"ptrMap"`
// }
