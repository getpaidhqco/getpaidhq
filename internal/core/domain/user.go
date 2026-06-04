package domain

// User represents an authenticated user/operator of an Org. Authentication
// itself is owned by adapters (Clerk, etc.); this type holds the persisted
// identity profile only.
type User struct {
	ID       uint
	Username string
	Email    string
}
