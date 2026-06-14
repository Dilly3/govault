package models

type User struct {
	Model
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"not null"` // 'customer' or 'admin'
}
