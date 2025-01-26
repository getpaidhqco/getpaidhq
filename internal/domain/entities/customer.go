package entities

type Customer struct {
	AccountId string `json:"acct_id"`
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}
