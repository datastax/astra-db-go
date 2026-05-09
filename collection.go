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

package astradb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/datastax/astra-db-go/cursors"
	"github.com/datastax/astra-db-go/filter"
	"github.com/datastax/astra-db-go/options"
	"github.com/datastax/astra-db-go/ptr"
	"github.com/datastax/astra-db-go/results"
	"github.com/datastax/astra-db-go/serdes"
	"github.com/datastax/astra-db-go/sort"
	"github.com/datastax/astra-db-go/table"
)

// CollectionFilter is implemented by [filter.F] and [filter.Filter].
// See the [filter package] for more details.
//
// Example composing Filters:
//
//	f := filter.Gt("num_pages", 300)
//
// Example using filter.F:
//
//	f := filter.F{"num_pages": filter.F{"$gt": 300}}
//
// [filter package]: https://pkg.go.dev/github.com/datastax/astra-db-go/filter
type CollectionFilter = filter.Filterable

// Collection represents a collection in an Astra DB database.
//
// Options set on the collection are inherited by all commands
// executed on it, unless overridden at the command level.
type Collection struct {
	db      *Db
	name    string
	options *options.APIOptions
}

// region Meta

// Name returns the collection name.
func (c *Collection) Name() string {
	return c.name
}

// ClientOptions returns the collection's options (or empty options if nil).
func (c *Collection) ClientOptions() *options.APIOptions {
	if c.options == nil {
		return &options.APIOptions{}
	}
	return c.options
}

// Database returns the parent database.
func (c *Collection) Database() *Db {
	return c.db
}

// newCmdWithMergedOptions creates a command with a pre-built *APIOptions cmdOpts,
// used by builder-pattern methods where API options flow through the struct.
func (c *Collection) newCmdWithMergedOptions(name string, payload any, cmdOpts *options.APIOptions) command {
	return newCmdWithMergedOptions(c.db, c.name, name, payload, c.options, serdes.TargetCollection, cmdOpts)
}

// resolveGeneralMethodTimeout returns the effective timeout for a paginated
// operation. The per-method timeout takes priority over the hierarchy timeout.
func (c *Collection) resolveGeneralMethodTimeout(methodTimeout *time.Duration, override *options.APIOptions) *time.Duration {
	if methodTimeout != nil {
		return methodTimeout
	}
	cmd := c.newCmdWithMergedOptions("", nil, override)
	opts := cmd.resolveOptions()
	return opts.GetGeneralMethodTimeout()
}

// endregion

// region Definition

// Options retrieves the collection's descriptor including its definition.
// This method calls the database's ListCollections and returns the descriptor
// for this specific collection.
//
// Options passed here override those set on the collection.
func (c *Collection) Options(ctx context.Context, opts ...options.CollectionOptionsOption) (*results.CollectionDescriptor, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}

	collections, err := c.db.ListCollections(ctx, &options.ListCollectionsOptions{APIOptions: merged.APIOptions})
	if err != nil {
		return nil, err
	}

	for _, coll := range collections {
		if coll.Name == c.name {
			return &coll, nil
		}
	}

	return nil, ErrNotFound
}

// endregion

// InsertOne inserts a single document into the collection.
//
// Options passed here override those set on the collection.
func (c *Collection) InsertOne(ctx context.Context, document any, opts ...options.CollectionInsertOneOption) (*results.InsertOneResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, fmt.Errorf("invalid options: %w", err)
	}
	return insertOne(ctx, document, c.newCmdWithMergedOptions, (*insertOneOptions)(merged), serdes.TargetCollection)
}

// InsertMany inserts documents into the collection. Param documents must be a non-empty slice.
//
// Options passed here override those set on the collection.
func (c *Collection) InsertMany(ctx context.Context, documents any, opts ...options.CollectionInsertManyOption) (*results.InsertManyResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return insertMany(ctx, documents, c.newCmdWithMergedOptions, (*insertManyOptions)(merged), serdes.TargetCollection)
}

// Payload for collection find one.
type collectionFindOnePayload struct {
	Filter     CollectionFilter          `json:"filter,omitempty"`
	Sort       sort.Sortable             `json:"sort,omitempty"`
	Projection map[string]any            `json:"projection,omitempty"`
	Options    *collectionFindOneOptions `json:"options,omitempty"`
}

// collectionFindOneOptions contains options for collection find one operations
type collectionFindOneOptions struct {
	IncludeSimilarity *bool `json:"includeSimilarity,omitempty"`
}

// FindOne finds a single document matching the filter.
//
// Options passed here override those set on the collection.
func (c *Collection) FindOne(ctx context.Context, f CollectionFilter, opts ...options.CollectionFindOneOption) *results.SingleResult {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return results.NewSingleResult([]byte{}, nil, nil, serdes.TargetCollection, err)
	}
	payload := collectionFindOnePayload{Filter: f, Sort: merged.Sort, Projection: merged.Projection}
	if merged.IncludeSimilarity != nil {
		payload.Options = &collectionFindOneOptions{IncludeSimilarity: merged.IncludeSimilarity}
	}
	cmd := c.newCmdWithMergedOptions("findOne", payload, merged.APIOptions)
	b, warnings, _, err := cmd.Execute(ctx)
	return results.NewSingleResult(b, warnings, nil, serdes.TargetCollection, err)
}

// Find returns a cursor for iterating over documents matching the filter.
//
// The cursor automatically handles pagination, fetching new pages as needed.
//
// The filter parameter defines criteria for selecting rows. Pass an empty filter.F{}
// or nil to find all rows (not recommended for large collections).
//
// Use options to specify sorting, projection, limits, and other behaviors.
//
// Example using Next/Decode pattern:
//
//	cursor := coll.Find(filter.F{"active": true})
//	defer cursor.Close()
//
//	for cursor.Next(ctx) {
//	    var doc MyDocument
//	    if err := cursor.Decode(&doc); err != nil {
//	        return err
//	    }
//	    // Process doc
//	}
//	if err := cursor.Err(); err != nil {
//	    return err
//	}
//
// Example getting all results at once:
//
//	cursor := coll.Find(filter.F{})
//	var docs []MyDocument
//	if err := cursor.DecodeAll(ctx, &docs); err != nil {
//	    return err
//	}
//
// Example with vector search:
//
//	cursor := coll.Find(filter.F{},
//	    options.CollectionFind().
//	        SetSort(map[string]any{"$vector": []float32{0.1, 0.2, 0.3}}).
//	        SetIncludeSimilarity(true),
//	)
//
// In the unlikely case of an option validation error while creating the cursor,
// the cursor will be returned in an unclearable errored state.
func (c *Collection) Find(f CollectionFilter, opts ...options.CollectionFindOption) *cursors.CollectionFindCursor {
	merged, err := options.MergeAndValidate(opts...)

	fetcher := func(ctx context.Context, payload any, opts *options.APIOptions) ([]byte, results.Warnings, *table.LazySchema, error) {
		cmd := c.newCmdWithMergedOptions("find", payload, merged.APIOptions)
		return cmd.Execute(ctx)
	}

	return cursors.NewCollectionFindCursor(f, merged, fetcher, err)
}

// collectionUpdateOnePayload is the payload for the updateOne command on collections.
type collectionUpdateOnePayload struct {
	Filter  CollectionFilter `json:"filter,omitempty"`
	Update  CollectionUpdate `json:"update"`
	Sort    sort.Sortable    `json:"sort,omitempty"`
	Options map[string]any   `json:"options,omitempty"`
}

// collectionUpdateResponse is the response from various update commands, where `MoreData` may or may not be present.
type collectionUpdateResponse struct {
	Status struct {
		MatchedCount  int  `json:"matchedCount"`
		ModifiedCount int  `json:"modifiedCount"`
		MoreData      bool `json:"moreData"`
		UpsertedId    any  `json:"upsertedId"`
	} `json:"status"`
}

// UpdateOne updates a single document matching the filter.
//
// The update parameter should be an [update.U] expression, e.g. update.Coll().Set("name", "new").
//
// Options passed here override those set on the collection.
func (c *Collection) UpdateOne(ctx context.Context, f CollectionFilter, u CollectionUpdate, opts ...options.CollectionUpdateOneOption) (*results.UpdateResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	payload := collectionUpdateOnePayload{
		Filter: f,
		Update: u,
		Sort:   merged.Sort,
	}

	if ptr.From(merged.Upsert) {
		payload.Options = map[string]any{"upsert": true}
	}

	cmd := c.newCmdWithMergedOptions("updateOne", payload, merged.APIOptions)
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	var resp collectionUpdateResponse
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection); err != nil {
		return nil, err
	}

	upsertedCount := 0
	if resp.Status.UpsertedId != nil {
		upsertedCount = 1
	}

	return &results.UpdateResult{
		MatchedCount:  resp.Status.MatchedCount,
		ModifiedCount: resp.Status.ModifiedCount,
		UpsertedCount: upsertedCount,
		UpsertedId:    resp.Status.UpsertedId,
	}, nil
}

// collectionUpdateManyPayload is the payload for the updateMany command on collections.
type collectionUpdateManyPayload struct {
	Filter  CollectionFilter `json:"filter,omitempty"`
	Update  CollectionUpdate `json:"update"`
	Options map[string]any   `json:"options,omitempty"`
}

// UpdateMany updates all documents matching the filter.
//
// The update parameter should be an [update.U] expression, e.g. update.Coll().Set("name", "new").
//
// The Data API may not update all matching documents in a single round-trip.
// This method automatically paginates, re-issuing the command and accumulating
// counts until the server indicates no more data remains.
//
// Options passed here override those set on the collection.
func (c *Collection) UpdateMany(ctx context.Context, f CollectionFilter, u CollectionUpdate, opts ...options.CollectionUpdateManyOption) (*results.UpdateResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	if timeout := c.resolveGeneralMethodTimeout(merged.Timeout, merged.APIOptions); timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	payload := collectionUpdateManyPayload{
		Filter: f,
		Update: u,
	}

	if ptr.From(merged.Upsert) {
		payload.Options = map[string]any{"upsert": true}
	}

	result := &results.UpdateResult{}

	for {
		cmd := c.newCmdWithMergedOptions("updateMany", payload, merged.APIOptions)
		b, _, _, err := cmd.Execute(ctx)
		if err != nil {
			return nil, err
		}

		var resp collectionUpdateResponse
		if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection); err != nil {
			return nil, err
		}

		result.MatchedCount += resp.Status.MatchedCount
		result.ModifiedCount += resp.Status.ModifiedCount

		if resp.Status.UpsertedId != nil && result.UpsertedId == nil {
			result.UpsertedId = resp.Status.UpsertedId
			result.UpsertedCount = 1
		}

		if !resp.Status.MoreData {
			break
		}
	}

	return result, nil
}

// collectionFindOneAndUpdatePayload is the payload for the findOneAndUpdate command.
type collectionFindOneAndUpdatePayload struct {
	Filter     CollectionFilter `json:"filter,omitempty"`
	Update     CollectionUpdate `json:"update"`
	Sort       sort.Sortable    `json:"sort,omitempty"`
	Projection map[string]any   `json:"projection,omitempty"`
	Options    map[string]any   `json:"options,omitempty"`
}

// FindOneAndUpdate finds a single document matching the filter, applies the update,
// and returns the document. By default, the document is returned as it was before the
// update. Use [options.ReturnDocumentAfter] to return the document after the update.
//
// The update parameter should be an [update.U] expression, e.g. update.Coll().Set("name", "new").
//
// Options passed here override those set on the collection.
func (c *Collection) FindOneAndUpdate(ctx context.Context, f CollectionFilter, u CollectionUpdate, opts ...options.CollectionFindOneAndUpdateOption) *results.SingleResult {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return results.NewSingleResult(nil, nil, nil, serdes.TargetCollection, err)
	}

	payload := collectionFindOneAndUpdatePayload{
		Filter:     f,
		Update:     u,
		Sort:       merged.Sort,
		Projection: merged.Projection,
	}

	payloadOpts := map[string]any{}
	if ptr.From(merged.Upsert) {
		payloadOpts["upsert"] = true
	}
	if merged.ReturnDocument != nil {
		payloadOpts["returnDocument"] = string(*merged.ReturnDocument)
	}
	if len(payloadOpts) > 0 {
		payload.Options = payloadOpts
	}

	cmd := c.newCmdWithMergedOptions("findOneAndUpdate", payload, merged.APIOptions)
	b, warnings, _, err := cmd.Execute(ctx)
	return results.NewSingleResult(b, warnings, nil, serdes.TargetCollection, err)
}

// ReplaceOne replaces a single document matching the filter.
//
// The replacement parameter should be a new document without an _id set.
//
// Options passed here override those set on the collection.
func (c *Collection) ReplaceOne(ctx context.Context, f CollectionFilter, replacement any, opts ...options.CollectionReplaceOneOption) (*results.UpdateResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	payload := collectionFindOneAndReplacePayload{
		Filter:      f,
		Replacement: replacement,
		Sort:        merged.Sort,
		Projection:  map[string]any{"*": 0},
	}

	if ptr.From(merged.Upsert) {
		payload.Options = map[string]any{"upsert": true}
	}

	cmd := c.newCmdWithMergedOptions("findOneAndReplace", payload, merged.APIOptions)
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	var resp collectionUpdateResponse
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, err
	}

	upsertedCount := 0
	if resp.Status.UpsertedId != nil {
		upsertedCount = 1
	}

	return &results.UpdateResult{
		MatchedCount:  resp.Status.MatchedCount,
		ModifiedCount: resp.Status.ModifiedCount,
		UpsertedCount: upsertedCount,
		UpsertedId:    resp.Status.UpsertedId,
	}, nil
}

// collectionFindOneAndReplacePayload is the payload for the findOneAndReplace command.
type collectionFindOneAndReplacePayload struct {
	Filter      CollectionFilter `json:"filter,omitempty"`
	Replacement any              `json:"replacement"`
	Sort        sort.Sortable    `json:"sort,omitempty"`
	Projection  map[string]any   `json:"projection,omitempty"`
	Options     map[string]any   `json:"options,omitempty"`
}

// FindOneAndReplace finds a single document matching the filter, replaces it,
// and returns the document. By default, the document is returned as it was before the
// replacement. Use [options.ReturnDocumentAfter] to return the document after the replacement.
//
// The replacement parameter should be a new document without an _id set.
//
// Options passed here override those set on the collection.
func (c *Collection) FindOneAndReplace(ctx context.Context, f CollectionFilter, replacement any, opts ...options.CollectionFindOneAndReplaceOption) *results.SingleResult {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return results.NewSingleResult(nil, nil, nil, serdes.TargetCollection, err)
	}

	payload := collectionFindOneAndReplacePayload{
		Filter:      f,
		Replacement: replacement,
		Sort:        merged.Sort,
		Projection:  merged.Projection,
	}

	payloadOpts := map[string]any{}
	if ptr.From(merged.Upsert) {
		payloadOpts["upsert"] = true
	}
	if merged.ReturnDocument != nil {
		payloadOpts["returnDocument"] = string(*merged.ReturnDocument)
	}
	if len(payloadOpts) > 0 {
		payload.Options = payloadOpts
	}

	cmd := c.newCmdWithMergedOptions("findOneAndReplace", payload, merged.APIOptions)
	b, warnings, _, err := cmd.Execute(ctx)
	return results.NewSingleResult(b, warnings, nil, serdes.TargetCollection, err)
}

// collectionDeleteOnePayload is the payload for the deleteOne command on collections.
type collectionDeleteOnePayload struct {
	Filter CollectionFilter `json:"filter,omitempty"`
	Sort   sort.Sortable    `json:"sort,omitempty"`
}

// collectionDeleteOneResponse is the response from the deleteOne command.
type collectionDeleteOneResponse struct {
	Status struct {
		DeletedCount int `json:"deletedCount"`
	} `json:"status"`
}

// DeleteOne deletes a single document matching the filter.
//
// When the filter matches multiple documents, use the Sort option to control
// which document is deleted.
//
// Options passed here override those set on the collection.
func (c *Collection) DeleteOne(ctx context.Context, f CollectionFilter, opts ...options.CollectionDeleteOneOption) (*results.DeleteResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	payload := collectionDeleteOnePayload{
		Filter: f,
		Sort:   merged.Sort,
	}

	cmd := c.newCmdWithMergedOptions("deleteOne", payload, merged.APIOptions)
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		return nil, err
	}

	var resp collectionDeleteOneResponse
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection); err != nil {
		return nil, err
	}

	return &results.DeleteResult{
		DeletedCount: resp.Status.DeletedCount,
	}, nil
}

// collectionDeleteManyPayload is the payload for the deleteMany command on collections.
type collectionDeleteManyPayload struct {
	Filter CollectionFilter `json:"filter,omitempty"`
}

// collectionDeleteManyResponse is the response from the deleteMany command.
type collectionDeleteManyResponse struct {
	Status struct {
		DeletedCount int  `json:"deletedCount"`
		MoreData     bool `json:"moreData"`
	} `json:"status"`
}

var ErrNilFilter = errors.New("filter cannot be nil. If you want to delete all documents, use an empty filter instead")

// DeleteMany deletes all documents matching the filter.
//
// The Data API may not delete all matching documents in a single round-trip.
// This method automatically paginates, re-issuing the command and accumulating
// counts until the server indicates no more data remains.
//
// An empty or nil filter deletes all documents in the collection. In that case,
// the returned DeletedCount is -1.
//
// Options passed here override those set on the collection.
func (c *Collection) DeleteMany(ctx context.Context, f CollectionFilter, opts ...options.CollectionDeleteManyOption) (*results.DeleteResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}

	if f == nil {
		return nil, ErrNilFilter
	}

	if timeout := c.resolveGeneralMethodTimeout(merged.Timeout, merged.APIOptions); timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	payload := collectionDeleteManyPayload{
		Filter: f,
	}

	result := &results.DeleteResult{}

	for {
		cmd := c.newCmdWithMergedOptions("deleteMany", payload, merged.APIOptions)
		b, _, _, err := cmd.Execute(ctx)
		if err != nil {
			return nil, err
		}

		var resp collectionDeleteManyResponse
		if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection); err != nil {
			return nil, err
		}

		result.DeletedCount += resp.Status.DeletedCount

		if !resp.Status.MoreData {
			break
		}
	}

	return result, nil
}

// collectionFindOneAndDeletePayload is the payload for the findOneAndDelete command.
type collectionFindOneAndDeletePayload struct {
	Filter     CollectionFilter `json:"filter,omitempty"`
	Sort       sort.Sortable    `json:"sort,omitempty"`
	Projection map[string]any   `json:"projection,omitempty"`
}

// FindOneAndDelete finds a single document matching the filter, deletes it,
// and returns the deleted document.
//
// Options passed here override those set on the collection.
func (c *Collection) FindOneAndDelete(ctx context.Context, f CollectionFilter, opts ...options.CollectionFindOneAndDeleteOption) *results.SingleResult {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return results.NewSingleResult(nil, nil, nil, serdes.TargetCollection, err)
	}

	payload := collectionFindOneAndDeletePayload{
		Filter:     f,
		Sort:       merged.Sort,
		Projection: merged.Projection,
	}

	cmd := c.newCmdWithMergedOptions("findOneAndDelete", payload, merged.APIOptions)
	b, warnings, _, err := cmd.Execute(ctx)
	return results.NewSingleResult(b, warnings, nil, serdes.TargetCollection, err)
}

// Payload for collection countDocuments.
type collectionCountPayload struct {
	Filter CollectionFilter `json:"filter,omitempty"`
}

type collectionCountResponse struct {
	Status struct {
		Count    int  `json:"count"`
		MoreData bool `json:"moreData"`
	} `json:"status"`
}

// CountDocuments counts documents after applying filter f. Count operations are
// expensive: for this reason, the best practice is to provide a reasonable upperBound.
//
// Options passed here override those set on the collection.
func (c *Collection) CountDocuments(ctx context.Context, f CollectionFilter, upperBound int, opts ...options.CollectionCountDocumentsOption) (int, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return 0, fmt.Errorf("invalid options: %w", err)
	}
	cmd := c.newCmdWithMergedOptions("countDocuments", collectionCountPayload{Filter: f}, merged.APIOptions)
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		return 0, err
	}

	var resp collectionCountResponse
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection); err != nil {
		return 0, err
	}

	// From the docs count all result:
	// > If the count exceeds the upper bound set by the API, then the status.count
	// > value will be the upper bound, and the status.moreData value is true.
	//
	// So - if we exceed what the API allows, we get an error. But we also enforce
	// the upper bound the user supplies. See also:
	// https://docs.datastax.com/en/astra-db-serverless/api-reference/document-methods/count-all.html#result
	if resp.Status.MoreData || resp.Status.Count > upperBound {
		return resp.Status.Count, results.ErrTooManyDocumentsToCount
	}
	return resp.Status.Count, nil
}

// EstimatedDocumentCount returns a rough estimate of the number of documents in the collection,
// much faster than CountDocuments but less precise. While it doesn't accept a filter, it can handle any
// number of documents, whereas CountDocuments may return an error if the count exceeds the upper bound.
//
// Options passed here override those set on the collection.
func (c *Collection) EstimatedDocumentCount(ctx context.Context, opts ...options.CollectionEstimatedDocumentCountOption) (int, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return 0, fmt.Errorf("invalid options: %w", err)
	}
	cmd := c.newCmdWithMergedOptions("estimatedDocumentCount", struct{}{}, merged.APIOptions)
	b, _, _, err := cmd.Execute(ctx)
	if err != nil {
		return 0, err
	}

	var resp collectionCountResponse // no "moreData" field in this response, but we can reuse the struct
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection); err != nil {
		return 0, err
	}

	return resp.Status.Count, nil
}

// Drop deletes the collection and all its documents. Use with caution.
func (c *Collection) Drop(ctx context.Context) error {
	return c.db.DropCollection(ctx, c.name)
}
