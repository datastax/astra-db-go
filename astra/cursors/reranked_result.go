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

package cursors

import "encoding/json"

// RerankedResult is a wrapper for a document returned by a findAndRerank operation,
// which also includes the scores calculated during the reranking process.
//
// Example usage with cursors.All:
//
//	for res, err := range cursors.All[cursors.RerankedResult[MyDoc]](ctx, cursor) {
//	    if err != nil { /* ... */ }
//	    fmt.Println(res.Document)
//	    fmt.Println(res.Scores)
//	}
//
// Alternatively, FindAndRerankCursor.GetScores may be used to access the scores for
// the current document without using a RerankedResult wrapper.
type RerankedResult[T any] struct {
	// Document is the decoded document.
	Document T

	// Scores contains the reranking scores for the document.
	Scores map[string]float32
}

// rawRerankedResult is used to buffer the documents and their scores before they are decoded into the user's result type.
type rawRerankedResult = RerankedResult[json.RawMessage]

// rerankedResultWrapper is used to identify RerankedResult[T] for decoding purposes
type rerankedResultWrapper interface {
	setScores(map[string]float32)
	documentAddr() any
}

func (r *RerankedResult[T]) setScores(scores map[string]float32) {
	r.Scores = scores
}

func (r *RerankedResult[T]) documentAddr() any {
	return &r.Document
}
