package handler

import (
	"time"

	"getpaidhq/internal/core/domain"
)

type SettingResponse struct {
	ParentId  string    `json:"parent_id"`
	Id        string    `json:"id"`
	Type      string    `json:"type"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewSettingResponse(s domain.Setting) SettingResponse {
	return SettingResponse{
		ParentId:  s.ParentId,
		Id:        s.Id,
		Type:      s.Type,
		Value:     s.Value,
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}
