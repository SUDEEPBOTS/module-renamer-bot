package database

import (
	"sync"
	"sync/atomic"
	"time"
)

type MemoryDatabase struct {
	mu           sync.RWMutex
	users        map[int64]User
	banned       map[int64]BannedUser
	totalRenames int64
}

func NewMemoryDatabase() *MemoryDatabase {
	return &MemoryDatabase{
		users:  make(map[int64]User),
		banned: make(map[int64]BannedUser),
	}
}

func (m *MemoryDatabase) AddUser(id int64, username, firstName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[id] = User{
		ID:        id,
		Username:  username,
		FirstName: firstName,
		JoinedAt:  time.Now().Unix(),
	}
	return nil
}

func (m *MemoryDatabase) GetUsers() ([]int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var list []int64
	for id := range m.users {
		list = append(list, id)
	}
	return list, nil
}

func (m *MemoryDatabase) CountUsers() (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int64(len(m.users)), nil
}

func (m *MemoryDatabase) IsBanned(id int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.banned[id]
	return exists
}

func (m *MemoryDatabase) BanUser(id int64, reason string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.banned[id] = BannedUser{
		ID:       id,
		Reason:   reason,
		BannedAt: time.Now().Unix(),
	}
	return nil
}

func (m *MemoryDatabase) UnbanUser(id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.banned, id)
	return nil
}

func (m *MemoryDatabase) IncrementRenames() error {
	atomic.AddInt64(&m.totalRenames, 1)
	return nil
}

func (m *MemoryDatabase) GetTotalRenames() int64 {
	return atomic.LoadInt64(&m.totalRenames)
}

func (m *MemoryDatabase) Close() error {
	return nil
}
