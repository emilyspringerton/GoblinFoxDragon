// Source [6] - Execution Engine
package chain

import (
	"sync"

	"github.com/dgraph-io/badger/v3"
)

type StateDB struct {
	db    *badger.DB
	cache map[string]*Account
	mu    sync.RWMutex
}

// See Source [17] for implementation details regarding NewStateDB and Commit
