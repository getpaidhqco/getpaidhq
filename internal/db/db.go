package db

import (
	"context"
)

type TransactionBeginner interface {
	Begin(ctx context.Context) (Committer, error)
}

type Committer interface {
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

type Database interface {
	TransactionBeginner
	Ping(ctx context.Context) error
	Close()
}
