package models

type Customer struct {
	AccountId string `json:"acct_id"`
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}
