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

package csrlike

import (
	"testing"
)

func TestCSRLike(t *testing.T) {
	t.Run("basic CSR structure", func(t *testing.T) {
		obj := &TestCSR{
			Username: "test-user",
			Extra: map[string]ExtraValue{
				"group":        {"system:authenticated", "developers"},
				"organization": {"acme-corp"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("empty extra", func(t *testing.T) {
		obj := &TestCSR{
			Username: "test-user",
			Extra:    map[string]ExtraValue{},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("nil extra", func(t *testing.T) {
		obj := &TestCSR{
			Username: "test-user",
			Extra:    nil,
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("update operations", func(t *testing.T) {
		oldObj := &TestCSR{
			Username: "test-user",
			Extra: map[string]ExtraValue{
				"group": {"system:authenticated"},
			},
		}
		newObj := &TestCSR{
			Username: "test-user",
			Extra: map[string]ExtraValue{
				"group":        {"system:authenticated", "developers"},
				"organization": {"acme-corp"},
			},
		}
		localSchemeBuilder.Test(t).Value(newObj).OldValue(oldObj).ExpectValid()
	})
}
