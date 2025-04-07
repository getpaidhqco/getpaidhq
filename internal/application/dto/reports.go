package dto

import "payloop/internal/domain/common"

type DataChangeEvent struct {
	Operation common.Operation
	Entity    common.Entity
	NewObject interface{}
	OldObject interface{}
}
