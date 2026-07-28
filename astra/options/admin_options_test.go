// Copyright IBM Corp.

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

package options_test

import (
	"testing"
	"time"

	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
)

func TestCreateDatabase_Defaults(t *testing.T) {
	opts, err := options.MergeAndValidate[options.CreateDatabaseOptions]()
	if err != nil {
		t.Fatalf("failed to merge options: %v", err)
	}
	if opts == nil {
		t.Fatal("expected non-nil options")
	}
	if opts.Blocking != nil {
		t.Errorf("expected Blocking to be nil on struct, got %v", opts.Blocking)
	}
	if opts.GetBlocking() != true {
		t.Errorf("expected GetBlocking() default true, got %v", opts.GetBlocking())
	}
	if opts.PollInterval != nil {
		t.Errorf("expected PollInterval to be nil on struct, got %v", opts.PollInterval)
	}
	if opts.GetPollInterval() != options.DefaultDatabasePollInterval {
		t.Errorf("expected GetPollInterval() default %v, got %v", options.DefaultDatabasePollInterval, opts.GetPollInterval())
	}
	if opts.Keyspace != nil {
		t.Errorf("expected Keyspace to be nil, got %v", opts.Keyspace)
	}
}

func TestDropDatabase_Defaults(t *testing.T) {
	opts, err := options.MergeAndValidate[options.DropDatabaseOptions]()
	if err != nil {
		t.Fatalf("failed to merge options: %v", err)
	}
	if opts.Blocking != nil {
		t.Errorf("expected Blocking to be nil on struct, got %v", opts.Blocking)
	}
	if opts.GetBlocking() != true {
		t.Errorf("expected GetBlocking() default true, got %v", opts.GetBlocking())
	}
	if opts.PollInterval != nil {
		t.Errorf("expected PollInterval to be nil on struct, got %v", opts.PollInterval)
	}
	if opts.GetPollInterval() != options.DefaultDatabasePollInterval {
		t.Errorf("expected GetPollInterval() default %v, got %v", options.DefaultDatabasePollInterval, opts.GetPollInterval())
	}
}

func TestCreateKeyspace_Defaults(t *testing.T) {
	opts, err := options.MergeAndValidate[options.CreateKeyspaceOptions]()
	if err != nil {
		t.Fatalf("failed to merge options: %v", err)
	}
	if opts.Blocking != nil {
		t.Errorf("expected Blocking to be nil on struct, got %v", opts.Blocking)
	}
	if opts.GetBlocking() != true {
		t.Errorf("expected GetBlocking() default true, got %v", opts.GetBlocking())
	}
	if opts.PollInterval != nil {
		t.Errorf("expected PollInterval to be nil on struct, got %v", opts.PollInterval)
	}
	if opts.GetPollInterval() != options.DefaultKeyspacePollInterval {
		t.Errorf("expected GetPollInterval() default %v, got %v", options.DefaultKeyspacePollInterval, opts.GetPollInterval())
	}
	if opts.Replication != nil {
		t.Errorf("expected ReplicationFactor to be nil, got %v", opts.Replication)
	}
}

func TestDropKeyspace_Defaults(t *testing.T) {
	opts, err := options.MergeAndValidate[options.DropKeyspaceOptions]()
	if err != nil {
		t.Fatalf("failed to merge options: %v", err)
	}
	if opts.Blocking != nil {
		t.Errorf("expected Blocking to be nil on struct, got %v", opts.Blocking)
	}
	if opts.GetBlocking() != true {
		t.Errorf("expected GetBlocking() default true, got %v", opts.GetBlocking())
	}
	if opts.PollInterval != nil {
		t.Errorf("expected PollInterval to be nil on struct, got %v", opts.PollInterval)
	}
	if opts.GetPollInterval() != options.DefaultKeyspacePollInterval {
		t.Errorf("expected GetPollInterval() default %v, got %v", options.DefaultKeyspacePollInterval, opts.GetPollInterval())
	}
}

func TestCreateDatabase_Override(t *testing.T) {
	opts, err := options.MergeAndValidate(
		options.CreateDatabase().SetBlocking(false).SetPollInterval(30 * time.Second),
	)
	if err != nil {
		t.Fatalf("failed to merge options: %v", err)
	}
	if opts == nil {
		t.Fatal("expected non-nil options")
	}
	if opts.Blocking == nil || *opts.Blocking != false {
		t.Errorf("expected Blocking to be false (user override), got %v", opts.Blocking)
	}
	if opts.PollInterval == nil || *opts.PollInterval != 30*time.Second {
		t.Errorf("expected PollInterval to be 30s (user override), got %v", opts.PollInterval)
	}
}

func TestCreateDatabase_OverrideAll(t *testing.T) {
	opts, err := options.MergeAndValidate(
		options.CreateDatabase().SetBlocking(false).SetPollInterval(45 * time.Second).SetKeyspace("my_keyspace"),
	)
	if err != nil {
		t.Fatalf("failed to merge options: %v", err)
	}
	if opts == nil {
		t.Fatal("expected non-nil options")
	}
	if opts.Blocking == nil || *opts.Blocking != false {
		t.Errorf("expected Blocking to be false, got %v", opts.Blocking)
	}
	if opts.PollInterval == nil || *opts.PollInterval != 45*time.Second {
		t.Errorf("expected PollInterval to be 45s, got %v", opts.PollInterval)
	}
	if opts.Keyspace == nil || *opts.Keyspace != "my_keyspace" {
		t.Errorf("expected Keyspace to be 'my_keyspace', got %v", opts.Keyspace)
	}
}

func TestDropKeyspace_MergeMultiple(t *testing.T) {
	first := options.DropKeyspace().
		SetBlocking(false).
		SetPollInterval(2 * time.Second)

	second := options.DropKeyspace().SetBlocking(true)

	third := options.DropKeyspace().SetPollInterval(5 * time.Second)

	opts, err := options.MergeAndValidate(first, second, third)
	if err != nil {
		t.Fatalf("failed to merge options: %v", err)
	}
	if opts == nil {
		t.Fatal("expected non-nil options")
	}

	if opts.GetBlocking() != true {
		t.Errorf("expected GetBlocking() to be true, got %v", opts.GetBlocking())
	}
	if opts.GetPollInterval() != 5*time.Second {
		t.Errorf("expected GetPollInterval() to be 5s, got %v", opts.GetPollInterval())
	}
}

func TestDropKeyspace_WithDirectStruct(t *testing.T) {
	// Make sure direct struct can be merged and that defaults are applied for missing fields
	opts, err := options.MergeAndValidate(
		&options.DropKeyspaceOptions{PollInterval: ptr.To(10 * time.Minute)},
	)
	if err != nil {
		t.Fatalf("failed to merge options: %v", err)
	}
	if opts == nil {
		t.Fatal("expected non-nil options")
	}
	if opts.GetBlocking() != true {
		t.Errorf("expected GetBlocking() to be true, got %v", opts.GetBlocking())
	}
	if opts.GetPollInterval() != 10*time.Minute {
		t.Errorf("expected GetPollInterval() to be 10m, got %v", opts.GetPollInterval())
	}
}

func TestMultipleDirectStruct(t *testing.T) {
	// Just wanted to create a test that combines multiple raw/direct struct instances
	// AND a flient builder.
	opts, err := options.MergeAndValidate(
		&options.DropKeyspaceOptions{PollInterval: ptr.To(1 * time.Minute)},
		&options.DropKeyspaceOptions{PollInterval: ptr.To(10 * time.Minute)},
		options.DropKeyspace().SetBlocking(false),
	)
	if err != nil {
		t.Fatalf("failed to merge options: %v", err)
	}
	if opts == nil {
		t.Fatal("expected non-nil options")
	}
	if opts.GetBlocking() != false {
		t.Errorf("expected GetBlocking() to be false, got %v", opts.GetBlocking())
	}
	// Now verify even with multiple direct structs, the last one wins for overlapping fields
	if opts.GetPollInterval() != 10*time.Minute {
		t.Errorf("expected GetPollInterval() to be 10m, got %v", opts.GetPollInterval())
	}
}
