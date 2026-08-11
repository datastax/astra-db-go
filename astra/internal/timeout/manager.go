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

package timeout

import (
	"context"
	"math"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/options"
)

// SingleType represents the type of timeout to use for single-call operations.
type SingleType int

const (
	GeneralMethod SingleType = iota
	CollectionAdmin
	TableAdmin
	KeyspaceAdmin
	DatabaseAdmin
)

// TimeoutManager manages timeouts for operations, tracking elapsed time
// and returning remaining time for each request.
type Manager struct {
	requestTimeout time.Duration
	overallTimeout time.Duration
	startTime      time.Time
	started        bool
}

func NewSingleCall(opts *options.APIOptions, tt SingleType) *Manager {
	requestTimeout := opts.GetRequestTimeout()
	var methodTimeout time.Duration

	switch tt {
	case CollectionAdmin:
		methodTimeout = opts.GetTimeout().GetCollectionAdmin()
	case TableAdmin:
		methodTimeout = opts.GetTimeout().GetTableAdmin()
	case KeyspaceAdmin:
		methodTimeout = opts.GetTimeout().GetKeyspaceAdmin()
	case DatabaseAdmin:
		methodTimeout = opts.GetTimeout().GetDatabaseAdmin()
	default: // GeneralMethod
		methodTimeout = opts.GetTimeout().GetGeneralMethod()
	}

	timeout := min(infIfZero(requestTimeout), infIfZero(methodTimeout))
	return &Manager{
		requestTimeout: timeout,
		overallTimeout: timeout,
	}
}

func NewMultiCall(opts ...options.APIOption) *Manager {
	resolved := options.Merge(opts...)
	return &Manager{
		requestTimeout: infIfZero(resolved.GetRequestTimeout()),
		overallTimeout: infIfZero(resolved.GetTimeout().GetGeneralMethod()),
	}
}

func infIfZero(d time.Duration) time.Duration {
	if d == 0 {
		return math.MaxInt64
	}
	return d
}

func (tm *Manager) Advance() time.Duration {
	if !tm.started {
		tm.startTime = time.Now()
		tm.started = true
	}

	if tm.requestTimeout == tm.overallTimeout {
		return tm.requestTimeout
	}

	elapsed := time.Since(tm.startTime)
	remaining := tm.overallTimeout - elapsed

	if remaining <= 0 {
		return 0
	}

	return min(remaining, tm.requestTimeout)
}

func (tm *Manager) ApplyToContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := tm.Advance()
	if timeout <= 0 {
		// Return a context that's already timed out
		return context.WithTimeout(ctx, 0)
	}
	return context.WithTimeout(ctx, timeout)
}
