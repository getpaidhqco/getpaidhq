package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/entities"
)

type Customer struct {
	OrgId          string                 `json:"org_id"`
	Id             string                 `json:"id"`
	FirstName      pgtype.Text            `json:"first_name"`
	LastName       pgtype.Text            `json:"last_name"`
	Email          pgtype.Text            `json:"email"`
	Phone          pgtype.Text            `json:"phone"`
	BillingAddress map[string]interface{} `json:"billing_address"`
	Metadata       map[string]string      `json:"metadata"`
	CreatedAt      pgtype.Date            `json:"created_at"`
	UpdatedAt      pgtype.Date            `json:"updated_at"`
}

func (c *Customer) ToEntity() entities.Customer {
	return entities.Customer{
		OrgId:          c.OrgId,
		Id:             c.Id,
		FirstName:      c.FirstName.String,
		LastName:       c.LastName.String,
		Email:          c.Email.String,
		Phone:          c.Phone.String,
		BillingAddress: entities.ParseAddress(c.BillingAddress),
		Metadata:       c.Metadata,
		CreatedAt:      c.CreatedAt.Time,
		UpdatedAt:      c.UpdatedAt.Time,
	}
}
