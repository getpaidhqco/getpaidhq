package port

import "context"

// TxManager runs fn inside a database transaction. The tx handle is
// propagated to repositories via ctx; callers must thread the ctx
// received in fn into any repo call that should participate in the tx.
// If fn returns nil, the tx is committed; if fn returns an error or
// panics, the tx is rolled back.
type TxManager interface {
	RunInTx(ctx context.Context, fn func(context.Context) error) error
}
