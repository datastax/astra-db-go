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

package options_test

import (
	"testing"

	"github.com/datastax/astra-db-go/options"
)

func TestMergeOptions_TypedNilBuilder(t *testing.T) {
	// A typed nil: the interface is non-nil but wraps a nil *DropKeyspaceOptionsBuilder.
	// This can happen when a caller conditionally assigns a builder variable.
	var nilBuilder *options.DropKeyspaceOptionsBuilder
	opts, err := options.MergeOptions(nilBuilder)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should still get defaults as if no options were passed.
	if opts.Blocking == nil || *opts.Blocking != true {
		t.Errorf("expected Blocking default true, got %v", opts.Blocking)
	}
	if opts.PollInterval == nil || *opts.PollInterval != options.DefaultKeyspacePollInterval {
		t.Errorf("expected PollInterval default, got %v", opts.PollInterval)
	}
}
