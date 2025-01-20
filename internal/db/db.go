package db

import "context"

type Database interface {
	GetDb() any
	Ping(ctx context.Context) error
	Close()
}
