package models

type Account struct {
	Model
	UserID   string `gorm:"index;not null"` // Link account to user
	Name     string
	Balance  int64
	Currency string `gorm:"not null"`
}
