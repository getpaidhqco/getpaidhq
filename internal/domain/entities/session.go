package entities

import "time"

type Session struct {
	OrgId     string    `json:"org_id"`
	Id        string    `json:"id"`
	CartId    string    `json:"cart_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
