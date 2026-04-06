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

package options

import "time"

// TODO: this is unused currently. Either figure out what my intent was when I created it or delete it.

// InsertOneOptions contains both Method options (sent to DB) and Request options (client side).
type InsertOneOptions struct {
	// Ordered controls whether the insert should be ordered.
	Ordered *bool `json:"ordered,omitempty"`

	// Timeout sets the timeout for the insert operation (client-side, not sent to the API).
	Timeout *time.Duration
}
