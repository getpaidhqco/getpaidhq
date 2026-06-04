package postgres

import "getpaidhq/internal/core/domain"

// userRow is the postgres on-the-wire shape of a User. Package-internal.
type userRow struct {
	ID       uint   `gorm:"column:id;primaryKey"`
	Username string `gorm:"column:name"`
	Email    string `gorm:"column:email;uniqueIndex"`
}

func (userRow) TableName() string { return "users" }

func (r userRow) toDomain() domain.User {
	return domain.User{
		ID:       r.ID,
		Username: r.Username,
		Email:    r.Email,
	}
}

func userRowFromDomain(u domain.User) userRow {
	return userRow{
		ID:       u.ID,
		Username: u.Username,
		Email:    u.Email,
	}
}
