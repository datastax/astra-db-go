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
	"github.com/datastax/astra-db-go/ptr"
)

func BenchmarkMergeOptions_WithValidator(b *testing.B) {
	// IndexingOptions has a real Validate() method.
	for b.Loop() {
		options.MergeAndValidate(options.Indexing().SetAllow("field1", "field2"))
	}
}

func BenchmarkMergeOptions_WithoutValidator(b *testing.B) {
	// CreateTableOptions has no Validate() method.
	for b.Loop() {
		options.MergeAndValidate(options.CreateTable().SetIfNotExists(true))
	}
}

func BenchmarkMergeOptions_NoopBuilder(b *testing.B) {
	// Raw struct via NoopBuilder path (reflection-based copy).
	for b.Loop() {
		opts := &options.CreateTableOptions{IfNotExists: ptr.To(true)}
		options.MergeAndValidate(opts)
	}
}

func BenchmarkMergeOptions_MultipleBuilders(b *testing.B) {
	// Multiple builders to measure per-option overhead.
	for b.Loop() {
		options.MergeAndValidate(
			options.CreateTable().SetIfNotExists(true),
			options.CreateTable().SetKeyspace("ks"),
		)
	}
}
