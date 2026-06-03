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

package tests

import (
	"context"
	"fmt"

	"github.com/datastax/astra-db-go/datatypes"
	"github.com/datastax/astra-db-go/filter"
	"github.com/datastax/astra-db-go/internal/integrationtests/harness"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/table"
)

func init() {
	harness.Register(
		harness.IntegrationTest{Name: "TableDuration", Run: TableDuration},
	)
}

const durationTableName = "go_test_duration"

type DurationRow struct {
	ID       string             `json:"id"`
	Label    string             `json:"label"`
	Duration datatypes.Duration `json:"duration"`
}

// TableDuration is self-contained: creates a table, exercises insert/find
// with Duration values, then drops the table.
func TableDuration(e *harness.TestEnv) error {
	ctx := context.Background()
	db := e.DefaultDb()

	def := table.Definition{
		Columns: table.Columns{
			{Name: "id", Column: table.Text()},
			{Name: "label", Column: table.Text()},
			{Name: "duration", Column: table.Column{Type: "duration"}},
		},
		PrimaryKey: table.PrimaryKey{
			PartitionBy: []string{"id"},
		},
	}

	tbl, err := db.CreateTable(ctx, durationTableName, def, options.CreateTable().SetIfNotExists(true))
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	defer func() {
		if dropErr := db.DropTable(ctx, durationTableName); dropErr != nil && err == nil {
			err = fmt.Errorf("drop table: %w", dropErr)
		}
	}()

	rows := []DurationRow{
		{ID: "zero", Label: "zero duration", Duration: datatypes.MustParseDuration("0s")},
		{ID: "simple", Label: "1 year 2 months", Duration: datatypes.MustParseDuration("1y2mo")},
		{ID: "days", Label: "3 weeks 4 days", Duration: datatypes.MustParseDuration("3w4d")},
		{ID: "time", Label: "5h6m7s", Duration: datatypes.MustParseDuration("5h6m7s")},
		{ID: "full", Label: "all units", Duration: datatypes.MustParseDuration("1y2mo3w4d5h6m7s8ms9us10ns")},
		{ID: "negative", Label: "negative", Duration: datatypes.MustParseDuration("-1y2mo")},
		{ID: "iso", Label: "ISO 8601", Duration: datatypes.MustParseDuration("P1Y2M3DT4H5M6S")},
	}

	if _, err = tbl.InsertMany(ctx, rows); err != nil {
		return fmt.Errorf("InsertMany: %w", err)
	}

	for _, want := range rows {
		var got DurationRow
		if err = tbl.FindOne(ctx, filter.Eq("id", want.ID)).Decode(&got); err != nil {
			return fmt.Errorf("FindOne(%q): %w", want.ID, err)
		}
		if !got.Duration.Equals(want.Duration) {
			return fmt.Errorf("row %q: duration mismatch: got %v, want %v", want.ID, got.Duration, want.Duration)
		}
	}

	return nil
}
