package banker

import (
	"log"

	"github.com/dilly3/govault/internal/store"
)

type Banker struct {
	storer store.Storer
	logger *log.Logger
}

func NewBanker(store store.Storer, logger *log.Logger) *Banker {
	return &Banker{storer: store, logger: logger}
}
