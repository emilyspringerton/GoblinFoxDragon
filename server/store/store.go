package store

import (
	"sync"
	"time"

	"dragonsnshit/packages/common"
)

type ClientState struct {
	Key     string
	LastCmd common.UserCmd
	Updated time.Time
}

type ClientStore interface {
	Upsert(key string, cmd common.UserCmd)
	Get(key string) (ClientState, bool)
	Delete(key string)
	All() []ClientState
}

type MemoryClientStore struct {
	mu      sync.RWMutex
	clients map[string]ClientState
	clock   func() time.Time
}

func NewMemoryClientStore() *MemoryClientStore {
	return &MemoryClientStore{
		clients: make(map[string]ClientState),
		clock:   time.Now,
	}
}

func (s *MemoryClientStore) Upsert(key string, cmd common.UserCmd) {
	if key == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[key] = ClientState{Key: key, LastCmd: cmd, Updated: s.clock()}
}

func (s *MemoryClientStore) Get(key string) (ClientState, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	client, ok := s.clients[key]
	return client, ok
}

func (s *MemoryClientStore) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, key)
}

func (s *MemoryClientStore) All() []ClientState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]ClientState, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	return clients
}
