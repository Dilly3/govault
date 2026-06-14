package store

import "gorm.io/gorm"

type Store struct {
	DB *gorm.DB
}

func NewStore(db *gorm.DB) *Store {
	return &Store{DB: db}
}

func (s *Store) GetDB() *gorm.DB {
	return s.DB
}

func (s *Store) GetUserStore() *UserStore {
	return &UserStore{DB: s.DB}
}

func (s *Store) GetAccountStore() *AccountStore {
	return &AccountStore{DB: s.DB}
}

func (s *Store) GetTransactionStore() *TransactionStore {
	return &TransactionStore{DB: s.DB}
}

func (s *Store) GetEntryStore() *EntryStore {
	return &EntryStore{DB: s.DB}
}
