package models

type Transaction struct {
	Model
	IdempotencyKey string `gorm:"uniqueIndex;not null"`
	Status         string `gorm:"not null"`
}
