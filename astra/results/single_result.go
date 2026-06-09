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

	"github.com/datastax/astra-db-go/v2/astra/serdes"
)

// SingleResult represents a document returned from an operation.
type SingleResult struct {
	err       error
	rawResp   []byte
	warnings  Warnings
	targetCtx serdes.TargetDecodeCtx
	target    serdes.Target
	desFlags  serdes.DesFlags
}

// NewSingleResult creates a new SingleResult with the given response, warnings, and error.
func NewSingleResult(rawResp []byte, warnings Warnings, targetCtx serdes.TargetDecodeCtx, target serdes.Target, err error, desFlags serdes.DesFlags) *SingleResult {
	return &SingleResult{
		rawResp:   rawResp,
		warnings:  warnings,
		targetCtx: targetCtx,
		target:    target,
		err:       err,
		desFlags:  desFlags,
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
	raw, err := sr.Raw()
	if err != nil {
		return err
	}
	return serdes.Deserialize(raw, v, sr.targetCtx, sr.target, sr.desFlags)
}

// Raw returns the raw JSON document from the API response.
// If no documents are found, returns [ErrNoDocuments].
func (sr *SingleResult) Raw() (json.RawMessage, error) {
	if sr.err != nil {
		return nil, sr.err
	}
	if len(sr.rawResp) == 0 {
		return nil, ErrNoDocuments
	}

	var srRaw singleResultJSON
	if err := serdes.Deserialize(sr.rawResp, &srRaw, nil, sr.target, sr.desFlags); err != nil {
		return nil, err
	}

	if string(srRaw.Data.Document) == "null" {
		return nil, ErrNoDocuments
	}
	return srRaw.Data.Document, nil
}

// Err returns any error that occurred during the operation that produced this [SingleResult].
func (sr *SingleResult) Err() error {
	return sr.err
}
