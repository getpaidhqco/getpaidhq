package entities

type Product struct {
	AccountId   string             `json:"account_id"`
	Id          string             `json:"id"`
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	Metadata    *map[string]string `json:"metadata"`
}
