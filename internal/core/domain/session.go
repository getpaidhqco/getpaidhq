package domain

import "time"

type Session struct {
	OrgId     string    `gorm:"column:org_id;primaryKey" json:"org_id"`
	Id        string    `gorm:"column:id;primaryKey" json:"id"`
	CartId    string    `gorm:"column:cart_id" json:"cart_id"`
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Session) TableName() string { return "sessions" }

type CreateSessionInput struct {
	OrgId    string            `json:"org_id"`
	Currency Currency          `json:"currency" binding:"required"`
	Country  string            `json:"country" binding:"required"`
	Metadata map[string]string `json:"metadata"`
}

type CreateSessionRequest struct {
	Currency Currency          `json:"currency" binding:"required"`
	Country  string            `json:"country" binding:"required"`
	Metadata map[string]string `json:"metadata"`
}

type CreateSessionResponse struct {
	Id     string `json:"id"`
	CartId string `json:"cart_id"`
}
