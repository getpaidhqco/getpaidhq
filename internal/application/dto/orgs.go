package dto

import "payloop/internal/api/authn"

type CreateOrgInput struct {
	Owner    authn.User        `json:"owner"`
	Name     string            `json:"name"`
	Country  string            `json:"country"`
	Timezone string            `json:"timezone"`
	Metadata map[string]string `json:"metadata"`
}
