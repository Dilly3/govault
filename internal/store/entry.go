package store

import "gorm.io/gorm"

type EntryStore struct {
	DB *gorm.DB
}

func NewEntryStore(db *gorm.DB) *EntryStore {
	return &EntryStore{DB: db}
}
