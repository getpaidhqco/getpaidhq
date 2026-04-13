package domain

type User struct {
	ID       uint   `gorm:"column:id;primaryKey" json:"id"`
	Username string `gorm:"column:name" json:"username"`
	Email    string `gorm:"column:email;uniqueIndex" json:"email"`
	Password string `gorm:"-" json:"password"`
}

func (User) TableName() string { return "users" }
