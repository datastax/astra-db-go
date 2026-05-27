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

// MultipleResult represents documents returned from an operation.
type MultipleResult struct {
	err      error
	rawResp  []byte
	warnings Warnings
}

// NewMultipleResult creates a new MultipleResult with the given response, warnings, and error.
func NewMultipleResult(rawResp []byte, warnings Warnings, err error) *MultipleResult {
	return &MultipleResult{
		rawResp:  rawResp,
		warnings: warnings,
		err:      err,
	}
}

// Warnings returns any warnings from the API response.
// Returns nil if there were no warnings.
func (mr *MultipleResult) Warnings() Warnings {
	return mr.warnings
}

// JSON returned from the astra API is in a format like this:
//
//	{ "data":{"documents":[ //... ], "nextPageState": "..."}}
type multipleResultJSON struct {
	Data struct {
		Documents     json.RawMessage `json:"documents"`
		NextPageState *string         `json:"nextPageState"`
	} `json:"data"`
}

// Decode will unmarshal the document represented by this [NewMultipleResult] into `v`.
// If no documents are found, returns [ErrNoDocuments].
func (mr *MultipleResult) Decode(v any) error {
	if mr.err != nil {
		return mr.err
	}
	if len(mr.rawResp) == 0 {
		return ErrNoDocuments
	}
	// First unmarshal to get rawmessage in data.document
	var result multipleResultJSON
	err := json.Unmarshal(mr.rawResp, &result)
	if err != nil {
		return err
	}
	// If document is null, that means we found no document
	if string(result.Data.Documents) == "null" {
		return ErrNoDocuments
	}
	// Then return/unmarshal the document
	return json.Unmarshal(result.Data.Documents, v)
}

// NextPageState returns the pagination state for fetching the next page of results.
// Returns nil if there are no more pages or if pagination is not supported for this query.
func (mr *MultipleResult) NextPageState() *string {
	if mr.err != nil || len(mr.rawResp) == 0 {
		return nil
	}
	var result multipleResultJSON
	err := json.Unmarshal(mr.rawResp, &result)
	if err != nil {
		return nil
	}
	return result.Data.NextPageState
}

// HasNextPage returns true if there are more pages of results available.
func (mr *MultipleResult) HasNextPage() bool {
	pageState := mr.NextPageState()
	return pageState != nil && *pageState != ""
}

// Error returns the error associated with this result, if any.
func (mr *MultipleResult) Error() error {
	return mr.err
}
