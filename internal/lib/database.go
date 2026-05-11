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
	GetClient() any
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

type DatabaseError struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e DatabaseError) Error() string {
	return e.Message
}

type ErrorCode string

const (
	NoResults           ErrorCode = "no_results"
	UniqueKeyViolation  ErrorCode = "unique_key_violation"
	NotNullViolation    ErrorCode = "not_null_violation"
	ForeignKeyViolation ErrorCode = "foreign_key_violation"
	UnknownTable        ErrorCode = "unknown_table"
	GenericError        ErrorCode = "generic"
)
