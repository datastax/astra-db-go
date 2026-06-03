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

package results

import (
	"encoding/json"
	"testing"

	"github.com/datastax/astra-db-go/astra/serdes"
)

func TestInsertManyError(t *testing.T) {
	// Create some dummy data
	apiErrors := DataAPIErrors{
		{Message: "failed to insert some"},
	}

	batches := []InsertManyBatch{
		{
			InsertedIds: []json.RawMessage{
				json.RawMessage(`"id1"`),
				json.RawMessage(`"id2"`),
			},
			TargetCtx: nil,
		},
	}

	res := NewInsertManyResult(batches, 2, nil, serdes.TargetNone, 0)

	err := &InsertManyError{
		Errors: apiErrors,
		Result: res,
	}

	if err.InsertedCount() != 2 {
		t.Errorf("expected 2 inserted docs, got %d", err.InsertedCount())
	}

	rawIds, _ := err.RawIDs()
	if len(rawIds) != 2 {
		t.Errorf("expected 2 raw ids, got %d", len(rawIds))
	}
	if string(rawIds[0]) != "\"id1\"" || string(rawIds[1]) != "\"id2\"" {
		t.Errorf("unexpected raw ids: %v", rawIds)
	}

	var decodedIds []string
	if decodeErr := err.DecodeIDs(&decodedIds); decodeErr != nil {
		t.Errorf("unexpected decode error: %v", decodeErr)
	}
	if len(decodedIds) != 2 || decodedIds[0] != "id1" || decodedIds[1] != "id2" {
		t.Errorf("unexpected decoded ids: %v", decodedIds)
	}

	expectedMsg := "insertMany failed after inserting 2 documents: failed to insert some"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}
