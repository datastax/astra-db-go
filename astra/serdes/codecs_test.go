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

package serdes

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"unsafe"
)

type RecursiveStruct struct {
	ID    int
	Ptr   *RecursiveStruct
	Slice []RecursiveStruct
	Map   map[string]*RecursiveStruct
}

type InterfaceStruct struct {
	Data any
}

func populateValue(v reflect.Value, depth int, seed int) {
	if depth > 3 {
		return
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int64:
		v.SetInt(int64(seed))
	case reflect.String:
		v.SetString(fmt.Sprintf("seed-%d", seed))
	case reflect.Ptr:
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		populateValue(v.Elem(), depth+1, seed)
	case reflect.Slice:
		slice := reflect.MakeSlice(v.Type(), 1, 1)
		populateValue(slice.Index(0), depth+1, seed)
		v.Set(slice)
	case reflect.Map:
		v.Set(reflect.MakeMap(v.Type()))
		key := reflect.ValueOf(fmt.Sprintf("k%d", seed))
		val := reflect.New(v.Type().Elem()).Elem()
		populateValue(val, depth+1, seed)
		v.SetMapIndex(key, val)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.CanSet() {
				populateValue(f, depth+1, seed+i)
			}
		}
	}
}

func TestCache_HeavyContention_Correctness(t *testing.T) {
	typeCodecs.Store(nil)

	var wg sync.WaitGroup
	numGoroutines := 64
	iterations := 100

	sharedTypes := []reflect.Type{
		reflect.TypeOf(RecursiveStruct{}),
		reflect.TypeOf(&RecursiveStruct{}),
		reflect.TypeOf(InterfaceStruct{}),
		reflect.TypeOf(map[string]RecursiveStruct{}),
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			ctx := codecCtx{}

			for j := 0; j < iterations; j++ {
				var typ reflect.Type
				if j%2 == 0 {
					typ = sharedTypes[j%len(sharedTypes)]
				} else {
					// Create unique dynamic types to force cache updates/CoW
					typ = reflect.StructOf([]reflect.StructField{
						{
							Name: fmt.Sprintf("Worker%dField%d", workerID, j),
							Type: reflect.TypeOf(""),
						},
					})
				}

				// 1. Resolve
				c := resolveCodecCaching(ctx, typ, false)
				if c.encode == nil || c.decode == nil {
					t.Errorf("worker %d: nil codec for type %s", workerID, typ)
					return
				}

				// 2. Setup Data
				valIn := reflect.New(typ).Elem()
				populateValue(valIn, 0, workerID+j)
				ptrIn := unsafe.Pointer(valIn.UnsafeAddr())

				// 3. Round-trip: Encode
				// Using empty context/target as per your signatures
				buf, err := c.encode(EncodeCtx{}, nil, ptrIn)
				if err != nil {
					t.Errorf("worker %d: encode failed: %v", workerID, err)
					continue
				}

				// 4. Round-trip: Decode
				valOut := reflect.New(typ).Elem()
				ptrOut := unsafe.Pointer(valOut.UnsafeAddr())
				_, err = c.decode(DecodeCtx{}, buf, ptrOut)
				if err != nil {
					t.Errorf("worker %d: decode failed: %v", workerID, err)
					continue
				}

				// 5. Verification
				// Comparing the Interface() values ensures the codec actually
				// mapped the bytes to the correct struct offsets.
				if !reflect.DeepEqual(valIn.Interface(), valOut.Interface()) {
					t.Errorf("worker %d: data corruption for type %s", workerID, typ)
				}
			}
		}(i)
	}

	wg.Wait()
}

func TestCache_ExactLength_SingleGoroutine(t *testing.T) {
	// Reset the cache for a clean state
	typeCodecs.Store(nil)

	ctx := codecCtx{}

	// 1. Define base types (4 types)
	sharedTypes := []reflect.Type{
		reflect.TypeOf(RecursiveStruct{}),
		reflect.TypeOf(&RecursiveStruct{}),
		reflect.TypeOf(InterfaceStruct{}),
		reflect.TypeOf(map[string]RecursiveStruct{}),
	}

	// 2. Resolve shared types
	for _, typ := range sharedTypes {
		resolveCodecCaching(ctx, typ, false)
	}

	// 3. Generate dynamic types (100 unique types)
	numDynamic := 100
	for i := 0; i < numDynamic; i++ {
		typ := reflect.StructOf([]reflect.StructField{
			{
				Name: fmt.Sprintf("UniqueField%d", i),
				Type: reflect.TypeOf(0),
			},
		})
		resolveCodecCaching(ctx, typ, false)
	}

	// 4. Calculate expected total
	// 4 (shared) + 100 (dynamic) = 104
	expectedTotal := len(sharedTypes) + numDynamic

	currentCache := cacheLoad()
	actualTotal := len(currentCache)

	if actualTotal != expectedTotal {
		t.Errorf("cache length mismatch: expected %d, got %d", expectedTotal, actualTotal)
	}

	// 5. Verify Idempotency (Should not increase count)
	for _, typ := range sharedTypes {
		resolveCodecCaching(ctx, typ, false)
	}

	if len(cacheLoad()) != expectedTotal {
		t.Errorf("cache size changed after resolving existing types: got %d", len(cacheLoad()))
	}
}
