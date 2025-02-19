package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Customer struct {
	OrgId     string      `json:"org_id"`
	Id        string      `json:"id"`
	Name      string      `json:"name"`
	Email     string      `json:"email"`
	Phone     string      `json:"phone"`
	CreatedAt pgtype.Date `json:"created_at"`
	UpdatedAt pgtype.Date `json:"updated_at"`
}

func (c *Customer) ToEntity() *entities.Customer {
	return &entities.Customer{
		OrgId:     c.OrgId,
		Id:        c.Id,
		Name:      c.Name,
		Email:     c.Email,
		Phone:     c.Phone,
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}
