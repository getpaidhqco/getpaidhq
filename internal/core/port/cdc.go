package port

import "context"

// CdcStreamHandler is the callback for processing change data capture events.
type CdcStreamHandler func(op string, entity string, newObj any, oldObj any)

// CdcStream provides a change data capture stream from the database.
type CdcStream interface {
	Start(ctx context.Context, handler CdcStreamHandler)
}
