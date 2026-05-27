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
	"fmt"
	"strings"
)

// Warning is duplicated here to avoid circular imports. It shares the same
// structure as DataAPIError.

// Warning represents a warning returned from the API.
// Warnings indicate non-fatal conditions that don't prevent the operation
// from completing, such as missing indexes.
type Warning struct {
	Message   string `json:"message"`
	ErrorCode string `json:"errorCode"`
	Family    string `json:"family"`
	Scope     string `json:"scope"`
	Title     string `json:"title"`
	ID        string `json:"id"`
}

// String implements fmt.Stringer for logging convenience.
func (w *Warning) String() string {
	if w == nil {
		return "<nil> Warning"
	}

	msg := w.Message
	if msg == "" {
		msg = "unknown warning"
	}

	var meta []string
	if w.ErrorCode != "" {
		meta = append(meta, fmt.Sprintf("code: %s", w.ErrorCode))
	}
	if w.Family != "" {
		meta = append(meta, fmt.Sprintf("family: %s", w.Family))
	}
	if w.Scope != "" {
		meta = append(meta, fmt.Sprintf("scope: %s", w.Scope))
	}

	if len(meta) > 0 {
		return fmt.Sprintf("%s (%s)", msg, strings.Join(meta, ", "))
	}

	return msg
}

// Warnings is a slice of warnings returned from API responses.
type Warnings []Warning
