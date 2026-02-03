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

// +k8s:validation-gen=TypeMeta
// +k8s:validation-gen-scheme-registry=k8s.io/code-generator/cmd/validation-gen/testscheme.Scheme

// Package mode tests the various modal validation patterns.
package mode

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

type StrictUnion struct {
	TypeMeta int

	// +k8s:discriminator
	Mode string `json:"mode"`

	// +k8s:member("A")=+k8s:required
	FieldA *string `json:"fieldA,omitempty"`

	// +k8s:member("B")=+k8s:required
	FieldB *string `json:"fieldB,omitempty"`
}

type SharedField struct {
	TypeMeta int

	// +k8s:discriminator
	Mode string `json:"mode"`

	// Valid in A and B, forbidden in C.
	// +k8s:member("A")=+k8s:optional
	// +k8s:member("B")=+k8s:optional
	Shared *string `json:"shared,omitempty"`
}

type ChainedValidation struct {
	TypeMeta int

	// +k8s:discriminator
	Mode string `json:"mode"`

	// In mode A, it is required AND must have maxLength 5.
	// +k8s:member("A")=+k8s:required
	// +k8s:member("A")=+k8s:maxLength=5
	Field *string `json:"field,omitempty"`
}

type ImplicitForbidden struct {
	TypeMeta int

	// +k8s:discriminator
	Mode string `json:"mode"`

	// Field is only mentioned for mode A. Mode B should implicitly forbid it.
	// +k8s:member("A")=+k8s:optional
	Field *string `json:"field,omitempty"`
}

type NonStringDiscriminator struct {
	TypeMeta int

	// +k8s:discriminator(name:"Bool")
	BoolMode bool `json:"boolMode"`

	// +k8s:member(discriminator:"Bool", value:"true")=+k8s:required
	BoolField *string `json:"boolField,omitempty"`

	// +k8s:discriminator(name:"Int")
	IntMode int `json:"intMode"`

	// +k8s:member(discriminator:"Int", value:"1")=+k8s:required
	IntField *string `json:"intField,omitempty"`
}

type MultipleDiscriminators struct {
	TypeMeta int

	// +k8s:discriminator(name:"Net")
	NetMode string `json:"netMode"`

	// +k8s:discriminator(name:"Storage")
	StorageMode string `json:"storageMode"`

	// +k8s:member(discriminator:"Net", value:"IPv6")=+k8s:required
	IPv6Config *string `json:"ipv6Config,omitempty"`

	// +k8s:member(discriminator:"Storage", value:"S3")=+k8s:required
	S3Bucket *string `json:"s3Bucket,omitempty"`
}

type ListTypeInsideMode struct {
	TypeMeta int

	// +k8s:discriminator
	Mode string `json:"mode"`

	// listType is technically a static property, but if specified inside a mode,
	// it should not trigger duplicate validations (one inside the mode, one global).
	// Because listType implementation relies on global state, this effectively
	// makes the list a map globally, but we verify here that we don't get
	// a redundant validation inside the modal block.
	//
	// +k8s:member("A")=+k8s:listType=map
	// +k8s:member("A")=+k8s:listMapKey=name
	Items []ListItem `json:"items,omitempty"`
}

type ListTypeSetInsideMode struct {
	TypeMeta int

	// +k8s:discriminator
	Mode string `json:"mode"`

	// +k8s:member("A")=+k8s:listType=set
	Items []string `json:"items,omitempty"`
}

type ListItem struct {
	Name string `json:"name"`
}

type TypeMeta int
