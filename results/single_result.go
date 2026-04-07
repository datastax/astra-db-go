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
)

// SingleResult represents a document returned from an operation.
type SingleResult struct {
	err      error
	rawResp  []byte
	warnings Warnings
}

// NewSingleResult creates a new SingleResult with the given response, warnings, and error.
func NewSingleResult(rawResp []byte, warnings Warnings, err error) *SingleResult {
	return &SingleResult{
		rawResp:  rawResp,
		warnings: warnings,
		err:      err,
	}
}

// Warnings returns any warnings from the API response.
// Returns nil if there were no warnings.
func (sr *SingleResult) Warnings() Warnings {
	return sr.warnings
}

// JSON returned from the astra API is in a format like this:
//
//	{ "data":{"document":{ //... }}}
type singleResultJSON struct {
	Data struct {
		Document json.RawMessage `json:"document"`
	} `json:"data"`
}

// Decode will unmarshal the document represented by this [SingleResult] into `v`.
// If no documents are found, returns [ErrNoDocuments].
func (sr *SingleResult) Decode(v any) error {
	if sr.err != nil {
		return sr.err
	}
	if len(sr.rawResp) == 0 {
		return ErrNoDocuments
	}
	// First unmarshal to get rawmessage in data.document
	var singleResult singleResultJSON
	err := json.Unmarshal(sr.rawResp, &singleResult)
	if err != nil {
		return err
	}
	// If document is null, that means we found no document
	if string(singleResult.Data.Document) == "null" {
		return ErrNoDocuments
	}
	// Then return/unmarshal the document
	return json.Unmarshal(singleResult.Data.Document, v)
}
