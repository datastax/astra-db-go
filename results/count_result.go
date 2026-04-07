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

// CountResult represents documents returned from an operation.
type CountResult struct {
	err      error
	rawResp  []byte
	warnings Warnings
}

// NewCountResult creates a new CountResult with the given response, warnings, and error.
func NewCountResult(rawResp []byte, warnings Warnings, err error) *CountResult {
	return &CountResult{
		rawResp:  rawResp,
		warnings: warnings,
		err:      err,
	}
}

// Warnings returns any warnings from the API response.
// Returns nil if there were no warnings.
func (cr *CountResult) Warnings() Warnings {
	return cr.warnings
}

// JSON returned from the astra API is in a format like this:
//
//	{ "status": { "moreData": true, "count": 1000 } }
//
// If there are too many rows, moreData will be true.
type CountResultJSON struct {
	Status struct {
		Count    int  `json:"count"`
		MoreData bool `json:"moreData"`
	} `json:"status"`
}

// Count returns the count as an int
func (mr *CountResult) Count(upperBound int) (int, error) {
	if mr.err != nil {
		return 0, mr.err
	}
	var resp CountResultJSON
	err := mr.Decode(&resp)
	// From the docs count all result:
	// > If the count exceeds the upper bound set by the API, then the status.count
	// > value will be the upper bound, and the status.moreData value is true.
	//
	// So - if we exceed what the API allows, we get an error. But we also enforce
	// the upper bound the user supplies. See also:
	// https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/count-all.html#result
	if resp.Status.MoreData ||
		(upperBound > 0 && resp.Status.Count > upperBound) {
		return resp.Status.Count, ErrTooManyDocumentsToCount
	}
	return resp.Status.Count, err
}

// Decode will unmarshal the document represented by this [NewCountResult] into `v`.
// If no documents are found, returns [ErrNoDocuments].
func (mr *CountResult) Decode(v *CountResultJSON) error {
	if mr.err != nil {
		return mr.err
	}
	if len(mr.rawResp) == 0 {
		return ErrNoDocuments
	}
	return json.Unmarshal(mr.rawResp, v)
}
