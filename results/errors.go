package results

import (
	"errors"
)

var ErrNoDocuments error = errors.New("no documents found")

// ErrTooManyDocumentsToCount is returned when a Count command exceeds the upper bounds.
var ErrTooManyDocumentsToCount error = errors.New("too many documents")
