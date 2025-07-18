package dto

import "payloop/internal/domain/common"

type CreateGatewayInput struct {
	OrgId    string            `json:"org_id" validate:"required"`
	PspId    common.Gateway    `json:"psp" validate:"required"`
	Name     string            `json:"name" `
	Settings map[string]string `json:"settings" validate:"required"`
}

type UpdateGatewayInput struct {
	OrgId    string            `json:"org_id" validate:"required"`
	Id       string            `json:"id" validate:"required"`
	Name     string            `json:"name" `
	Settings map[string]string `json:"settings" validate:"required"`
}

type GatewayFilter struct {
	OrgId string `json:"org_id" validate:"required"`
}
