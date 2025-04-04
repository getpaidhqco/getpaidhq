package models

import (
	"github.com/jackc/pgx/v5/pgtype"
	"payloop/internal/domain/common"
	"payloop/internal/domain/entities"
	"payloop/internal/domain/entities/payment_methods"
	"time"
)

type PaymentMethod struct {
	OrgId          string
	Id             string
	Status         string
	Psp            string
	Name           string
	CustomerId     string
	BillingAddress map[string]interface{}
	Type           string
	Token          string
	Metadata       map[string]string
	Details        interface{}
	ExpireAt       pgtype.Date
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (s *PaymentMethod) ToEntity() entities.PaymentMethod {

	return entities.PaymentMethod{
		OrgId:          s.OrgId,
		Id:             s.Id,
		Status:         payment_methods.PaymentMethodStatus(s.Status),
		Psp:            s.Psp,
		Name:           s.Name,
		CustomerId:     s.CustomerId,
		BillingAddress: entities.ParseAddress(s.BillingAddress),
		Type:           payment_methods.PaymentMethodType(s.Type),
		Token:          s.Token,
		Details:        s.Details,
		ExpireAt:       s.ExpireAt.Time,
		Metadata:       s.Metadata,
		CreatedAt:      s.CreatedAt,
		UpdatedAt:      s.UpdatedAt,
	}
}

type Address struct {
	FirstName  pgtype.Text
	LastName   pgtype.Text
	Email      pgtype.Text
	Phone      pgtype.Text
	Line1      pgtype.Text
	Line2      pgtype.Text
	City       pgtype.Text
	State      pgtype.Text
	PostalCode pgtype.Text
	Country    pgtype.Text
}

func (a *Address) ToEntity() entities.Address {
	return entities.Address{
		FirstName:  a.FirstName.String,
		LastName:   a.LastName.String,
		Email:      a.Email.String,
		Phone:      a.Phone.String,
		Line1:      a.Line1.String,
		Line2:      a.Line2.String,
		City:       a.City.String,
		State:      a.State.String,
		PostalCode: a.PostalCode.String,
		Country:    common.Country(a.Country.String),
	}
}
