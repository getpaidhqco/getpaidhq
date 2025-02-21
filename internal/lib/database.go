package lib

import (
	"context"
)

const (
	// DBTransaction is database transaction handle set at router context
	DBTransaction = "db_trx"
)

type TransactionBeginner interface {
	Begin(ctx context.Context) (Committer, error)
}

type Committer interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	GetClient() interface{}
}

type Database interface {
	TransactionBeginner
	Ping(ctx context.Context) error
	Close()
}

type Tx interface {
	Begin(ctx context.Context) (Tx, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
