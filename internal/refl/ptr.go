// Copyright IBM Corp.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package refl

import (
	"reflect"
	"unsafe"
)

// UnwindPointerType recursively unwraps pointer types, returning the underlying type.
func UnwindPointerType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

type IFace struct {
	Typ unsafe.Pointer
	Ptr unsafe.Pointer
}

type TypeID unsafe.Pointer

// GetTypeID extracts the runtime type-descriptor pointer from a reflect.Type interface value.
// Each distinct Go type has a unique, stable *abi.Type at a fixed address, so this gives us
// a cheap, hashable identity — the same approach used by serdes/codecs.go.
func GetTypeID(t reflect.Type) TypeID {
	return TypeID((*IFace)(unsafe.Pointer(&t)).Ptr)
}

// GetValuePtr cheaply extracts the underlying pointer from an addressable reflect.Value interface value.
func GetValuePtr(v reflect.Value) unsafe.Pointer {
	return (*IFace)(unsafe.Pointer(&v)).Ptr
}
