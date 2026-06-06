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

// Warning represents a warning returned from the API.
// Warnings indicate non-fatal conditions that don't prevent the operation
// from completing, such as missing indexes.
type Warning DataAPIError

// String implements fmt.Stringer for logging convenience.
func (w *Warning) String() string {
	if w == nil {
		return "<nil> Warning"
	}

	// Cast to DataAPIError to use its Error() method
	msg := (*DataAPIError)(w).Error()

	// Error() might return "<nil> DataAPIError" if message is empty and no meta,
	// but DataAPIError.Error() handles empty messages by returning "unknown data api error".
	// However, Warning should probably say "unknown warning" if it falls through.
	if msg == "unknown data api error" {
		return "unknown warning"
	}

	return msg
}

// Error implements the error interface for Warning.
func (w Warning) Error() string {
	return w.String()
}

// Warnings is a slice of warnings returned from API responses.
type Warnings []Warning
