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

// InsertOneOptions contains both Method options (sent to DB) and Request options (client side).
type InsertOneOptions struct {
	// Method Options (sent in JSON)
	Ordered *bool `json:"ordered,omitempty"`

	// Request Options (handled by client)
	Timeout *time.Duration
}

// List implements Builder[InsertOneOptions] allowing the raw struct to be
// passed directly to methods that accept ...Builder[InsertOneOptions].
func (o *InsertOneOptions) List() []func(*InsertOneOptions) {
	return NoopBuilder(o)
}

// Validate implements Validator for InsertOneOptions.
func (o InsertOneOptions) Validate() error { return nil }

// InsertOneOptionsBuilder is a builder for InsertOneOptions.
type InsertOneOptionsBuilder struct {
	Opts []func(*InsertOneOptions)
}

// InsertOne creates a new InsertOneOptionsBuilder.
func InsertOne() *InsertOneOptionsBuilder {
	return &InsertOneOptionsBuilder{}
}

// List implements Builder[InsertOneOptions].
func (b *InsertOneOptionsBuilder) List() []func(*InsertOneOptions) {
	return b.Opts
}

// SetOrdered sets whether the insert should be ordered.
func (b *InsertOneOptionsBuilder) SetOrdered(v bool) *InsertOneOptionsBuilder {
	b.Opts = append(b.Opts, func(o *InsertOneOptions) { o.Ordered = &v })
	return b
}

// SetTimeout sets the timeout for the insert operation.
func (b *InsertOneOptionsBuilder) SetTimeout(v time.Duration) *InsertOneOptionsBuilder {
	b.Opts = append(b.Opts, func(o *InsertOneOptions) { o.Timeout = &v })
	return b
}
