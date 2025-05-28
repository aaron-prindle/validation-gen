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

// This is a test package for maps of slices with all primitive types.
package mapofslicesprimitives

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

type TestStruct struct {
	TypeMeta int

	// String types
	// +k8s:eachVal=+k8s:eachVal=+k8s:maxLength=10
	StringSlices map[string][]string `json:"stringSlices"`

	// Numeric types
	// +k8s:eachVal=+k8s:eachVal=+k8s:minimum=0
	IntSlices   map[string][]int   `json:"intSlices"`
	Int8Slices  map[string][]int8  `json:"int8Slices"`
	Int16Slices map[string][]int16 `json:"int16Slices"`
	Int32Slices map[string][]int32 `json:"int32Slices"`
	Int64Slices map[string][]int64 `json:"int64Slices"`

	// +k8s:eachVal=+k8s:eachVal=+k8s:minimum=0
	UintSlices   map[string][]uint   `json:"uintSlices"`
	Uint8Slices  map[string][]uint8  `json:"uint8Slices"`
	Uint16Slices map[string][]uint16 `json:"uint16Slices"`
	Uint32Slices map[string][]uint32 `json:"uint32Slices"`
	Uint64Slices map[string][]uint64 `json:"uint64Slices"`

	// Float types
	Float32Slices map[string][]float32 `json:"float32Slices"`
	Float64Slices map[string][]float64 `json:"float64Slices"`

	// Boolean type
	BoolSlices map[string][]bool `json:"boolSlices"`

	// Byte slices - special case, allowed as [][]byte
	ByteSlices map[string][]byte `json:"byteSlices"`
	
	// Maps of slices of byte slices - also allowed
	ByteSliceSlices map[string][][]byte `json:"byteSliceSlices"`
}