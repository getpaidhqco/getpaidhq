package models

type Session struct {
	AccountId string `json:"account_id"`
	Id        string `json:"id"`
	CartId    string `json:"cart_id"`
}
