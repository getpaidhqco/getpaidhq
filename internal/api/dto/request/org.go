package request

type CreateOrgInput struct {
	Name     string            `json:"name" binding:"required"`
	Country  string            `json:"country" binding:"required"`
	Timezone string            `json:"timezone" binding:"required"`
	Metadata map[string]string `json:"metadata"`
}
