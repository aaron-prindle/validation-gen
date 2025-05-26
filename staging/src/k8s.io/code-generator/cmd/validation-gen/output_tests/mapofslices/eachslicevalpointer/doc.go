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

// This is a test package for slices with structs containing pointer fields.
package eachslicevalpointer

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

// SimpleStruct is a basic struct type
type SimpleStruct struct {
	// +k8s:maxLength=20
	Name string `json:"name"`
	// +k8s:minimum=0
	Value int `json:"value"`
}

type TestStruct struct {
	TypeMeta            int
	StructSlices        []SimpleStruct  `json:"structSlices"`
	ComplexStructSlices []ComplexStruct `json:"complexStructSlices"`
	StringSlices        []string        `json:"stringSlices"`

	// +k8s:eachVal=+k8s:maxLength=15
	ValidatedStringSlices []string `json:"validatedStringSlices"`
}

type ComplexStruct struct {
	OptionalName *string `json:"optionalName,omitempty"`
	OptionalInt  *int    `json:"optionalInt,omitempty"`

	// Nested struct pointer - this is where the nil pointer issue occurs
	// +k8s:optional
	Nested *SimpleStruct `json:"nested,omitempty"`
}
