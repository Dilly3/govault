package store

import "gorm.io/gorm"

type Storer interface {
	GetDB() *gorm.DB
	GetUserStore() *UserStore
	GetAccountStore() *AccountStore
	GetTransactionStore() *TransactionStore
	GetEntryStore() *EntryStore
}
