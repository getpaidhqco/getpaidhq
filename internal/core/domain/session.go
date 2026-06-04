package domain

import "time"

// Session is a short-lived checkout context tied to an Org and a Cart.
type Session struct {
	OrgId     string
	Id        string
	CartId    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
