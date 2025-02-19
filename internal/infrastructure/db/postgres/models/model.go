package models

type Model interface {
	ToEntity() interface{}
}
