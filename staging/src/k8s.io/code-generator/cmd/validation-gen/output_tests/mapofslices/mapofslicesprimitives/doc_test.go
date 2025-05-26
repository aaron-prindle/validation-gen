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

package mapofslicesprimitives

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestMapOfSlicesPrimitives(t *testing.T) {
	t.Run("string validation", func(t *testing.T) {
		obj := &TestStruct{
			StringSlices: map[string][]string{
				"valid": {"short", "ok"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("string too long", func(t *testing.T) {
		obj := &TestStruct{
			StringSlices: map[string][]string{
				"invalid": {"this-string-is-too-long"},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(field.NewPath("stringSlices").Key("invalid").Index(0), "this-string-is-too-long", "must be no more than 10 bytes"),
		)
	})

	t.Run("integer validation", func(t *testing.T) {
		obj := &TestStruct{
			IntSlices:   map[string][]int{"nums": {0, 1, 2, 3}},
			Int8Slices:  map[string][]int8{"small": {0, 1, 2}},
			Int16Slices: map[string][]int16{"medium": {0, 100, 200}},
			Int32Slices: map[string][]int32{"large": {0, 1000, 2000}},
			Int64Slices: map[string][]int64{"xlarge": {0, 10000, 20000}},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("negative integer", func(t *testing.T) {
		obj := &TestStruct{
			IntSlices: map[string][]int{"nums": {-1, 0, 1}},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectInvalid(
			field.Invalid(field.NewPath("intSlices").Key("nums").Index(0), -1, "must be greater than or equal to 0"),
		)
	})

	t.Run("unsigned integer validation", func(t *testing.T) {
		obj := &TestStruct{
			UintSlices:   map[string][]uint{"nums": {0, 1, 2, 3}},
			Uint8Slices:  map[string][]uint8{"bytes": {0, 255}},
			Uint16Slices: map[string][]uint16{"words": {0, 65535}},
			Uint32Slices: map[string][]uint32{"dwords": {0, 4294967295}},
			Uint64Slices: map[string][]uint64{"qwords": {0, 18446744073709551615}},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("float validation", func(t *testing.T) {
		obj := &TestStruct{
			Float32Slices: map[string][]float32{"floats": {0.0, 1.5, 3.14}},
			Float64Slices: map[string][]float64{"doubles": {0.0, 1.5, 3.14159265359}},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("boolean validation", func(t *testing.T) {
		obj := &TestStruct{
			BoolSlices: map[string][]bool{
				"flags":  {true, false, true},
				"states": {false, false},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("byte slices", func(t *testing.T) {
		obj := &TestStruct{
			ByteSlices: map[string][]byte{
				"data1": []byte("hello"),
				"data2": []byte{0x00, 0xFF, 0x42},
			},
			ByteSliceSlices: map[string][][]byte{
				"multi": {[]byte("hello"), []byte("world")},
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("all types together", func(t *testing.T) {
		obj := &TestStruct{
			StringSlices:    map[string][]string{"s": {"test"}},
			IntSlices:       map[string][]int{"i": {1, 2}},
			UintSlices:      map[string][]uint{"u": {1, 2}},
			Float32Slices:   map[string][]float32{"f": {1.0}},
			BoolSlices:      map[string][]bool{"b": {true}},
			ByteSlices:      map[string][]byte{"by": []byte("data")},
			ByteSliceSlices: map[string][][]byte{"bys": {[]byte("a"), []byte("b")}},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})

	t.Run("empty and nil cases", func(t *testing.T) {
		obj := &TestStruct{
			StringSlices: map[string][]string{
				"empty": {},
				"nil":   nil,
			},
			IntSlices: map[string][]int{
				"empty": {},
				"nil":   nil,
			},
		}
		localSchemeBuilder.Test(t).Value(obj).ExpectValid()
	})
}
