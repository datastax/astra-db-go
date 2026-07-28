// Copyright DataStax, Inc.
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

package timeout

import (
	"context"
	"testing"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
)

func TestSingleCallTimeoutManager(t *testing.T) {
	opts := &options.APIOptions{
		Timeout: &options.TimeoutOptions{
			Request:       ptr.To(10 * time.Second),
			GeneralMethod: ptr.To(20 * time.Second),
		},
	}
	tm := NewSingleCall(opts, GeneralMethod)

	timeout := tm.Advance()
	if timeout != 10*time.Second {
		t.Errorf("expected 10s (min of 10s and 20s), got %v", timeout)
	}

	// Second call should return same timeout
	timeout = tm.Advance()
	if timeout != 10*time.Second {
		t.Errorf("expected 10s on second call, got %v", timeout)
	}
}

func TestSingleCallTimeoutManager_MethodShorter(t *testing.T) {
	opts := &options.APIOptions{
		Timeout: &options.TimeoutOptions{
			Request:       ptr.To(20 * time.Second),
			GeneralMethod: ptr.To(10 * time.Second),
		},
	}
	tm := NewSingleCall(opts, GeneralMethod)

	timeout := tm.Advance()
	if timeout != 10*time.Second {
		t.Errorf("expected 10s (min of 20s and 10s), got %v", timeout)
	}
}

func TestMultiCallTimeoutManager(t *testing.T) {
	opts := &options.APIOptions{
		Timeout: &options.TimeoutOptions{
			Request:       ptr.To(5 * time.Second),
			GeneralMethod: ptr.To(15 * time.Second),
		},
	}
	tm := NewMultiCall(opts)

	timeout1 := tm.Advance()
	if timeout1 != 5*time.Second {
		t.Errorf("expected 5s for first call, got %v", timeout1)
	}

	time.Sleep(100 * time.Millisecond)

	timeout2 := tm.Advance()
	if timeout2 < 4*time.Second || timeout2 > 5*time.Second {
		t.Errorf("expected ~5s for second call, got %v", timeout2)
	}
}

func TestMultiCallTimeoutManager_RemainingLessThanRequest(t *testing.T) {
	opts := &options.APIOptions{
		Timeout: &options.TimeoutOptions{
			Request:       ptr.To(10 * time.Second),
			GeneralMethod: ptr.To(500 * time.Millisecond),
		},
	}
	tm := NewMultiCall(opts)

	timeout1 := tm.Advance()
	if timeout1 < 400*time.Millisecond || timeout1 > 500*time.Millisecond {
		t.Errorf("expected ~500ms for first call, got %v", timeout1)
	}

	time.Sleep(400 * time.Millisecond)

	timeout2 := tm.Advance()
	if timeout2 > 200*time.Millisecond {
		t.Errorf("expected remaining time < 200ms, got %v", timeout2)
	}
}

func TestTimeoutManager_ApplyToContext(t *testing.T) {
	opts := &options.APIOptions{
		Timeout: &options.TimeoutOptions{
			Request:       ptr.To(1 * time.Second),
			GeneralMethod: ptr.To(2 * time.Second),
		},
	}
	tm := NewSingleCall(opts, GeneralMethod)

	ctx := context.Background()
	newCtx, cancel := tm.ApplyToContext(ctx)
	defer cancel()

	deadline, ok := newCtx.Deadline()
	if !ok {
		t.Fatal("expected context to have deadline")
	}

	remaining := time.Until(deadline)
	if remaining < 900*time.Millisecond || remaining > 1100*time.Millisecond {
		t.Errorf("expected ~1s deadline, got %v", remaining)
	}
}
