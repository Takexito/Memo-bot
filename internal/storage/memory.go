package storage

import (
	"sync"
	"time"
)

type threadInfo struct {
	ThreadID   string
	CreatedAt  time.Time
	LastUsedAt time.Time
}

type MemoryStorage struct {
	mu       sync.RWMutex
	lastID   int64
	userMeta map[int64]*UserMetadata
	threads  map[int64]threadInfo
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		lastID:   0,
		userMeta: make(map[int64]*UserMetadata),
		threads:  make(map[int64]threadInfo),
	}
}

func (s *MemoryStorage) GetUserMetadata(userID int64) (*UserMetadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if meta, exists := s.userMeta[userID]; exists {
		return meta, nil
	}

	// Initialize new metadata if not exists
	return &UserMetadata{
		UserID:     userID,
		LastUsedAt: time.Now(),
	}, nil
}

func (s *MemoryStorage) UpdateUserMetadata(metadata *UserMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	metadata.LastUsedAt = time.Now()
	s.userMeta[metadata.UserID] = metadata
	return nil
}

func (s *MemoryStorage) AddUserCategory(userID int64, category string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, exists := s.userMeta[userID]
	if !exists {
		meta = &UserMetadata{
			UserID:     userID,
			Categories: []string{},
			Tags:       []string{},
			LastUsedAt: time.Now(),
		}
	}

	// Check if category already exists
	for _, c := range meta.Categories {
		if c == category {
			return nil
		}
	}

	meta.Categories = append(meta.Categories, category)
	meta.LastUsedAt = time.Now()
	s.userMeta[userID] = meta
	return nil
}

func (s *MemoryStorage) AddUserTag(userID int64, tag string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	meta, exists := s.userMeta[userID]
	if !exists {
		meta = &UserMetadata{
			UserID:     userID,
			Categories: []string{},
			Tags:       []string{},
			LastUsedAt: time.Now(),
		}
	}

	// Check if tag already exists
	for _, t := range meta.Tags {
		if t == tag {
			return nil
		}
	}

	meta.Tags = append(meta.Tags, tag)
	meta.LastUsedAt = time.Now()
	s.userMeta[userID] = meta
	return nil
}

func (s *MemoryStorage) Close() error {
	// Nothing to close for in-memory storage
	return nil
}

func (s *MemoryStorage) GetThread(userID int64) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if thread, exists := s.threads[userID]; exists {
		return thread.ThreadID, nil
	}
	return "", nil
}

func (s *MemoryStorage) SaveThread(userID int64, threadID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.threads[userID] = threadInfo{
		ThreadID:   threadID,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
	}
	return nil
}

func (s *MemoryStorage) UpdateThreadLastUsed(userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if thread, exists := s.threads[userID]; exists {
		thread.LastUsedAt = time.Now()
		s.threads[userID] = thread
	}
	return nil
}

func (s *MemoryStorage) DeleteThread(userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.threads, userID)
	return nil
}
