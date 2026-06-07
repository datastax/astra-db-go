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

package astra

import (
	"context"
	"testing"

	"github.com/datastax/astra-db-go/astra/internal/command"
	"github.com/datastax/astra-db-go/astra/options"
	"github.com/datastax/astra-db-go/astra/results"
	"github.com/datastax/astra-db-go/astra/serdes"
)

// Example response when your application is resuming
const resumingResponse = "{\"message\":\"Your database is resuming from hibernation and will be available in the next few minutes.\"}"

func TestCommandDBResuming(t *testing.T) {
	cmd := command.DataAPI{Endpoint: "http://localhost"}
	_, _, _, err := cmd.ExtractErrors(503, []byte(resumingResponse), nil)
	if err == nil {
		t.Error("Expected error but got none")
	}
}

const projectionSchemaResponse = "{\"data\":{\"document\":{\"int\":1,\"text\":\"1\"}},\"status\":{\"projectionSchema\":{\"text\":{\"type\":\"text\"},\"int\":{\"type\":\"int\"},\"ascii\":{\"type\":\"ascii\"},\"bigint\":{\"type\":\"bigint\"},\"blob\":{\"type\":\"blob\"},\"boolean\":{\"type\":\"boolean\"},\"date\":{\"type\":\"date\"},\"decimal\":{\"type\":\"decimal\"},\"double\":{\"type\":\"double\"},\"duration\":{\"type\":\"duration\"},\"float\":{\"type\":\"float\"},\"inet\":{\"type\":\"inet\"},\"list\":{\"type\":\"list\",\"valueType\":{\"type\":\"userDefined\",\"udtName\":\"example_udt\",\"definition\":{\"fields\":{\"name\":{\"type\":\"text\"},\"age\":{\"type\":\"varint\"},\"id\":{\"type\":\"uuid\"}}}}},\"map\":{\"type\":\"map\",\"keyType\":\"varint\",\"valueType\":{\"type\":\"userDefined\",\"udtName\":\"example_udt\",\"definition\":{\"fields\":{\"name\":{\"type\":\"text\"},\"age\":{\"type\":\"varint\"},\"id\":{\"type\":\"uuid\"}}}}},\"set\":{\"type\":\"set\",\"valueType\":\"uuid\"},\"smallint\":{\"type\":\"smallint\"},\"time\":{\"type\":\"time\"},\"timestamp\":{\"type\":\"timestamp\"},\"tinyint\":{\"type\":\"tinyint\"},\"udt\":{\"type\":\"userDefined\",\"udtName\":\"example_udt\",\"definition\":{\"fields\":{\"name\":{\"type\":\"text\"},\"age\":{\"type\":\"varint\"},\"id\":{\"type\":\"uuid\"}}},\"apiSupport\":{\"createTable\":true,\"insert\":true,\"read\":true,\"filter\":false,\"cqlDefinition\":\"default_keyspace.example_udt\"}},\"uuid\":{\"type\":\"uuid\"},\"varint\":{\"type\":\"varint\"},\"vector\":{\"type\":\"vector\",\"dimension\":5,\"apiSupport\":{\"createTable\":true,\"insert\":true,\"read\":true,\"filter\":false,\"cqlDefinition\":\"VECTOR<float,5>\"}}}}}"

func TestCommandWithProjectionSchema(t *testing.T) {
	cmd := command.DataAPI{Endpoint: "http://localhost"}
	_, _, _, err := cmd.ExtractErrors(200, []byte(projectionSchemaResponse), nil)
	if err != nil {
		t.Errorf("Did not expect error but got: %v", err)
	}
}

// Example response when already exists
const createAlreadyExistsResponse = "{\"status\":{\"insertedIds\":[]},\"errors\":[{\"message\":\"Document already exists with the given _id\",\"errorCode\":\"DOCUMENT_ALREADY_EXISTS\",\"id\":\"4055f085-68d8-4c2d-8d91-90a0722b5fef\",\"title\":\"Document already exists with the given _id\",\"family\":\"REQUEST\",\"scope\":\"DOCUMENT\"}]}"

func TestCommandAlreadyExistsErr(t *testing.T) {
	cmd := command.DataAPI{Endpoint: "http://localhost"}
	_, _, _, err := cmd.ExtractErrors(200, []byte(createAlreadyExistsResponse), nil)
	t.Logf("err value:\n%s", err)
	if err == nil {
		t.Error("Expected error but got none")
	}
}

// Example response with warnings (truncated for brevity in test)
const warningsResponse = `{"data":{"documents":[]},"status":{"warnings":[{"errorCode":"WARN1","message":"Warning 1"}]}}`

func TestCommandWarnings(t *testing.T) {
	cmd := command.DataAPI{Endpoint: "http://localhost"}
	_, warnings, _, err := cmd.ExtractErrors(200, []byte(warningsResponse), nil)
	if err != nil {
		t.Errorf("Did not expect error but got: %v", err)
	}
	if len(warnings) != 1 {
		t.Errorf("Expected 1 warning but got: %d", len(warnings))
	}
}

func TestMarshalJSONWithName(t *testing.T) {
	cmd := command.DataAPI{
		Name:    "createCollection",
		Payload: map[string]any{"name": "my_collection"},
	}
	got, err := serdes.Serialize(cmd, serdes.TargetNone, serdes.SortMapKeys)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"createCollection":{"name":"my_collection"}}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestMarshalJSONWithoutName(t *testing.T) {
	cmd := command.DataAPI{
		Payload: map[string]string{"key": "value"},
	}
	got, err := serdes.Serialize(cmd, serdes.TargetNone, serdes.SortMapKeys)
	if err != nil {
		t.Fatal(err)
	}
	expected := `{"key":"value"}`
	if string(got) != expected {
		t.Errorf("expected %s, got %s", expected, string(got))
	}
}

func TestExtractErrorsWarningHandler(t *testing.T) {
	var called int
	handler := func(w results.Warning) {
		called++
	}
	opts := options.Merge(options.API().SetWarningHandler(handler))

	cmd := command.DataAPI{Endpoint: "http://localhost"}
	_, _, _, err := cmd.ExtractErrors(200, []byte(warningsResponse), opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called != 1 {
		t.Errorf("expected warning handler called 1 time, got %d", called)
	}
}

func TestURLDatabaseAdmin(t *testing.T) {
	id, region := "db-id", "us-east-1"
	client := NewClient()
	db := newDbFromID(id, region, options.AstraEnvironmentProd, client, nil)
	cmd := db.newAdminCmd("findKeyspaces", nil, nil)
	got, err := cmd.URL()
	if err != nil {
		t.Fatal(err)
	}
	expected := "https://db-id-us-east-1.apps.astra.datastax.com/api/json/v1"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestURLNonAstraBackend(t *testing.T) {
	hcd := options.DataAPIBackendHCD
	client := NewClient()
	db := newDbFromEndpoint("http://localhost:8181", client, options.Join(nil, options.API().SetDataAPIBackend(hcd)))
	cmd := db.Collection("my_collection").newCmd("find", nil, nil)

	got, err := cmd.URL()
	if err != nil {
		t.Fatal(err)
	}
	// Non-astra: no "api/json" prefix, just version/keyspace/resource
	expected := "http://localhost:8181/v1/default_keyspace/my_collection"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestCommandOptionsHierarchy(t *testing.T) {
	// Test that command correctly merges:
	// Client -> DB -> Collection -> Command Builders -> Command Struct
	client := NewClient(options.API().SetToken("client-token"))
	db := client.Database("http://localhost:8181", options.API().SetKeyspace("db-keyspace"))
	coll := db.Collection("my-coll", options.API().SetAPIVersion("v2"))

	// Create command with both builders and a merged struct override
	cmd := coll.newCmd("find", nil, options.API().SetHeader("X-Custom", "value"))
	cmd.Options = options.Join(cmd.Options, options.API().SetToken("final-token"))

	opts := cmd.ResolveOptions()

	if token, _ := opts.TokenProvider.Token(context.Background()); token != "final-token" {
		t.Errorf("expected final-token, got %q", token)
	}
	if opts.GetKeyspace() != "db-keyspace" {
		t.Errorf("expected db-keyspace, got %q", opts.GetKeyspace())
	}
	if opts.GetAPIVersion() != "v2" {
		t.Errorf("expected v2, got %q", opts.GetAPIVersion())
	}
	if opts.Headers["X-Custom"] != "value" {
		t.Errorf("expected header value, got %q", opts.Headers["X-Custom"])
	}
}
