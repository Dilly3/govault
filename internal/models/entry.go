package models

type Entry struct {
	Model
	TransactionID uint  `gorm:"index;not null"`
	AccountID     uint  `gorm:"index;not null"`
	Amount        int64 // Negative for debit, Positive for credit
}
