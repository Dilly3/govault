package store

import (
	"gorm.io/gorm"
)

type TransactionStore struct {
	DB *gorm.DB
}

func NewTransactionStore(db *gorm.DB) *TransactionStore {
	return &TransactionStore{DB: db}
}
