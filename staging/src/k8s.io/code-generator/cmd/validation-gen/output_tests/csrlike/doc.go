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

// This is a test package mimicking CSR structure.
package csrlike

import "k8s.io/code-generator/cmd/validation-gen/testscheme"

var localSchemeBuilder = testscheme.New()

// ExtraValue represents extra information about a user.
type ExtraValue []string

// TestCSR mimics CertificateSigningRequest structure
type TestCSR struct {
	TypeMeta int

	// Username is the requesting user
	Username string `json:"username"`

	// Extra contains extra attributes of the user that created the request.
	// +optional
	Extra map[string]ExtraValue `json:"extra,omitempty"`
}
