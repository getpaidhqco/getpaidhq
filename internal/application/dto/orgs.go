package dto

type CreateOrgInput struct {
	Name     string            `json:"name"`
	Country  string            `json:"country"`
	Timezone string            `json:"timezone"`
	Metadata map[string]string `json:"metadata"`
}
