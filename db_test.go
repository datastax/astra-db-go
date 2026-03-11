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

package astradb

import (
	"encoding/json"
	"testing"

	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/ptr"
)

func TestCreateCollectionCommand(t *testing.T) {
	t.Run("no options", func(t *testing.T) {
		cmd, err := createCollectionCommand(getTestDb(t), "my_collection")
		if err != nil {
			t.Fatalf("createCollectionCommand: %v", err)
		}
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		// Should not have "options" key when no options provided
		expected := `{"createCollection":{"name":"my_collection"}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", expected, string(cmdBytes))
		}
	})

	t.Run("with vector", func(t *testing.T) {
		cmd, err := createCollectionCommand(getTestDb(t), "my_collection",
			options.CreateCollection().SetVector(&options.VectorOptions{
				Dimension: ptr.To(1024),
				Metric:    ptr.To("cosine"),
			}))
		if err != nil {
			t.Fatalf("createCollectionCommand: %v", err)
		}
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		expected := `{"createCollection":{"name":"my_collection","options":{"vector":{"dimension":1024,"metric":"cosine"}}}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", expected, string(cmdBytes))
		}
	})

	t.Run("multiple builders merged", func(t *testing.T) {
		// Later option should override earlier
		cmd, err := createCollectionCommand(getTestDb(t), "my_collection",
			options.CreateCollection().SetVector(&options.VectorOptions{
				Dimension: ptr.To(512),
				Metric:    ptr.To("euclidean"),
			}),
			options.CreateCollection().SetVector(&options.VectorOptions{
				Dimension: ptr.To(1024),
				Metric:    ptr.To("cosine"),
			}),
		)
		if err != nil {
			t.Fatalf("createCollectionCommand: %v", err)
		}
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		// Last-write-wins: dimension=1024, metric=cosine
		expected := `{"createCollection":{"name":"my_collection","options":{"vector":{"dimension":1024,"metric":"cosine"}}}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", expected, string(cmdBytes))
		}
	})

	t.Run("raw struct passed directly", func(t *testing.T) {
		rawOpts := &options.CreateCollectionOptions{
			DefaultId: &options.CollectionDefaultIdOptions{
				Type: ptr.To(options.DefaultIdTypeUUIDv7),
			},
		}
		cmd, err := createCollectionCommand(getTestDb(t), "my_collection", rawOpts)
		if err != nil {
			t.Fatalf("createCollectionCommand: %v", err)
		}
		cmdBytes, err := json.Marshal(cmd)
		if err != nil {
			t.Fatalf("json.Marshal: %v", err)
		}
		expected := `{"createCollection":{"name":"my_collection","options":{"defaultId":{"type":"uuidv7"}}}}`
		if string(cmdBytes) != expected {
			t.Errorf("expected JSON:\n%s\nGot:\n%s", expected, string(cmdBytes))
		}
	})
}
