package store

import "gorm.io/gorm"

type AccountStore struct {
	DB *gorm.DB
}

func NewAccountStore(db *gorm.DB) *AccountStore {
	return &AccountStore{DB: db}
}
