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
	"errors"

	"github.com/datastax/astra-db-go/v2/astra/cursors"
	"github.com/datastax/astra-db-go/v2/astra/filter"
	"github.com/datastax/astra-db-go/v2/astra/internal/command"
	"github.com/datastax/astra-db-go/v2/astra/internal/timeout"
	"github.com/datastax/astra-db-go/v2/astra/options"
	"github.com/datastax/astra-db-go/v2/astra/ptr"
	"github.com/datastax/astra-db-go/v2/astra/results"
	"github.com/datastax/astra-db-go/v2/astra/serdes"
	"github.com/datastax/astra-db-go/v2/astra/update"
	"github.com/datastax/astra-db-go/v2/internal/utils"
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
// [filter package]: https://pkg.go.dev/github.com/datastax/astra-db-go/v2/astra/filter
type CollectionFilter = filter.Filterable

// CollectionUpdate is implemented by [update.CollectionUpdateBuilder] and [update.U].
// See the [update package] for more details.
//
// [update package]: https://pkg.go.dev/github.com/datastax/astra-db-go/v2/astra/update
type CollectionUpdate = update.CollectionUpdate

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

// ClientOptions returns the collection's options as a resolved struct with defaults.
func (c *Collection) ClientOptions() *options.APIOptions {
	return c.options
}

// Database returns the parent database.
func (c *Collection) Database() *Db {
	return c.db
}

// newCmd creates a command for this collection.
func (c *Collection) newCmd(name string, payload any, opts ...options.APIOption) command.DataAPI {
	return command.NewDataAPICommand(c.db.endpoint, c.name, name, payload, serdes.TargetCollection, options.Merge(append([]options.APIOption{c.options}, opts...)...))
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
		return nil, err
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

// region Insertions

// InsertOne inserts a single document into the collection.
//
// Options passed here override those set on the collection.
func (c *Collection) InsertOne(ctx context.Context, document any, opts ...options.CollectionInsertOneOption) (*results.InsertOneResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return insertOne(ctx, document, c.newCmd, (insertOneOptions)(*merged), serdes.TargetCollection)
}

// InsertMany inserts documents into the collection. Param documents must be a non-empty slice.
//
// Options passed here override those set on the collection.
func (c *Collection) InsertMany(ctx context.Context, documents any, opts ...options.CollectionInsertManyOption) (*results.InsertManyResult, error) {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return nil, err
	}
	return insertMany(ctx, documents, c.options, c.newCmd, (insertManyOptions)(*merged), serdes.TargetCollection)
}

// endregion

// region Finds

// FindOne finds a single document matching the filter.
//
// Options passed here override those set on the collection.
func (c *Collection) FindOne(ctx context.Context, f CollectionFilter, opts ...options.CollectionFindOneOption) *results.SingleResult {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return results.NewSingleResult(nil, nil, nil, serdes.TargetCollection, err, c.ClientOptions().GetDesFlags())
	}
	return findOne(ctx, f, c.newCmd, (findOneOptions)(*merged), serdes.TargetCollection)
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

	fetcher := func(ctx context.Context, payload any, opts *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
		cmd := c.newCmd("find", payload, merged.APIOptions)
		return cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	}

	return cursors.NewCollectionFindCursor(f, merged, fetcher, err)
}

// FindAndRerank returns a cursor for iterating over documents returned by a collection findAndRerank operation.
//
// The cursor automatically handles pagination, fetching new pages as needed.
//
// Use options to specify sorting, projection, limits, and other behaviors.
//
// Example using Next/Decode pattern:
//
//	cursor := coll.FindAndRerank(filter.F{"active": true})
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
// In the unlikely case of an option validation error while creating the cursor,
// the cursor will be returned in an unclearable errored state.
func (c *Collection) FindAndRerank(f CollectionFilter, opts ...options.CollectionFindAndRerankOption) cursors.FindAndRerankCursor {
	merged, err := options.MergeAndValidate(opts...)

	fetcher := func(ctx context.Context, payload any, opts *options.APIOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
		cmd := c.newCmd("findAndRerank", payload, merged.APIOptions)
		return cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	}

	return cursors.NewCollectionFindAndRerankCursor(f, merged, fetcher, err)
}

// endregion

// region Updates

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
	b, err := updateOne(ctx, f, u, c.newCmd, (updateOneOptions)(*merged))

	var resp collectionUpdateResponse
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection, merged.APIOptions.GetDesFlags()); err != nil {
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

	// Create timeout manager for multi-call operation
	tm := timeout.NewMultiCall(c.options, merged.APIOptions)

	payload := map[string]any{
		"filter": f,
		"update": u,
		"options": map[string]any{
			"upsert": ptr.From(merged.Upsert),
		},
	}

	result := &results.UpdateResult{}

	for {
		cmd := c.newCmd("updateMany", payload, merged.APIOptions)
		b, _, _, err := cmd.Execute(ctx, tm)
		if err != nil {
			return nil, err
		}

		var resp collectionUpdateResponse
		if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection, merged.APIOptions.GetDesFlags()); err != nil {
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
		return results.NewSingleResult(nil, nil, nil, serdes.TargetCollection, err, c.ClientOptions().GetDesFlags())
	}

	cmd := c.newCmd("findOneAndUpdate", map[string]any{
		"filter":     f,
		"update":     u,
		"sort":       merged.Sort,
		"projection": utils.NonNilMap(merged.Projection),
		"options": map[string]any{
			"upsert":         merged.Upsert,
			"returnDocument": merged.ReturnDocument,
		},
	}, merged.APIOptions)

	b, warnings, schema, err := cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	return results.NewSingleResult(b, warnings, schema, serdes.TargetCollection, err, merged.APIOptions.GetDesFlags())
}

// endregion

// region Replacements

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

	b, _, _, err := c.findOneAndReplace(ctx, f, replacement, &options.CollectionFindOneAndReplaceOptions{
		Sort:       merged.Sort,
		Projection: map[string]any{"*": 0},
		Upsert:     merged.Upsert,
		APIOptions: merged.APIOptions,
	})

	var resp collectionUpdateResponse
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection, merged.APIOptions.GetDesFlags()); err != nil {
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
		return results.NewSingleResult(nil, nil, nil, serdes.TargetCollection, err, c.ClientOptions().GetDesFlags())
	}

	b, warnings, schema, err := c.findOneAndReplace(ctx, f, replacement, &options.CollectionFindOneAndReplaceOptions{
		Sort:           merged.Sort,
		Projection:     merged.Projection,
		Upsert:         merged.Upsert,
		ReturnDocument: merged.ReturnDocument,
		APIOptions:     merged.APIOptions,
	})

	return results.NewSingleResult(b, warnings, schema, serdes.TargetCollection, err, merged.APIOptions.GetDesFlags())
}

func (c *Collection) findOneAndReplace(ctx context.Context, f CollectionFilter, replacement any, opts *options.CollectionFindOneAndReplaceOptions) ([]byte, results.Warnings, serdes.TargetDecodeCtx, error) {
	cmd := c.newCmd("findOneAndReplace", map[string]any{
		"filter":      f,
		"replacement": replacement,
		"sort":        opts.Sort,
		"projection":  utils.NonNilMap(opts.Projection),
		"options": map[string]any{
			"upsert":         opts.Upsert,
			"returnDocument": opts.ReturnDocument,
		},
	}, opts.APIOptions)

	return cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
}

// endregion

// region Deletions

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

	b, err := deleteOne(ctx, f, c.newCmd, (deleteOneOptions)(*merged))
	if err != nil {
		return nil, err
	}

	var resp collectionDeleteOneResponse
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection, merged.APIOptions.GetDesFlags()); err != nil {
		return nil, err
	}

	return &results.DeleteResult{
		DeletedCount: resp.Status.DeletedCount,
	}, nil
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

	tm := timeout.NewMultiCall(c.options, merged.APIOptions)

	payload := map[string]any{
		"filter": f,
	}

	result := &results.DeleteResult{}

	for {
		cmd := c.newCmd("deleteMany", payload, merged.APIOptions)
		b, _, _, err := cmd.Execute(ctx, tm)
		if err != nil {
			return nil, err
		}

		var resp collectionDeleteManyResponse
		if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection, merged.APIOptions.GetDesFlags()); err != nil {
			return nil, err
		}

		result.DeletedCount += resp.Status.DeletedCount

		if !resp.Status.MoreData {
			break
		}
	}

	return result, nil
}

// FindOneAndDelete finds a single document matching the filter, deletes it,
// and returns the deleted document.
//
// Options passed here override those set on the collection.
func (c *Collection) FindOneAndDelete(ctx context.Context, f CollectionFilter, opts ...options.CollectionFindOneAndDeleteOption) *results.SingleResult {
	merged, err := options.MergeAndValidate(opts...)
	if err != nil {
		return results.NewSingleResult(nil, nil, nil, serdes.TargetCollection, err, 0)
	}

	cmd := c.newCmd("findOneAndDelete", map[string]any{
		"filter":     f,
		"sort":       merged.Sort,
		"projection": utils.NonNilMap(merged.Projection),
	}, merged.APIOptions)

	b, warnings, schema, err := cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	return results.NewSingleResult(b, warnings, schema, serdes.TargetCollection, err, merged.APIOptions.GetDesFlags())
}

// endregion

// region Counts

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
		return 0, err
	}

	cmd := c.newCmd("countDocuments", map[string]any{"filter": f}, merged.APIOptions)
	b, _, _, err := cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	if err != nil {
		return 0, err
	}

	var resp collectionCountResponse
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection, merged.APIOptions.GetDesFlags()); err != nil {
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
		return 0, err
	}

	cmd := c.newCmd("estimatedDocumentCount", struct{}{}, merged.APIOptions)
	b, _, _, err := cmd.ExecuteSingle(ctx, timeout.GeneralMethod)
	if err != nil {
		return 0, err
	}

	var resp collectionCountResponse // no "moreData" field in this response, but we can reuse the struct
	if err := serdes.Deserialize(b, &resp, nil, serdes.TargetCollection, merged.APIOptions.GetDesFlags()); err != nil {
		return 0, err
	}

	return resp.Status.Count, nil
}

// endregion

// region Misc

// Drop deletes the collection and all its documents. Use with caution.
func (c *Collection) Drop(ctx context.Context, opts ...options.DropCollectionOption) error {
	return c.db.DropCollection(ctx, c.name, opts...)
}

// endregion
