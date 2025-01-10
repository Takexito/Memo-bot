package storage

import (
    "context"
    "sort"
    "sync"
    "time"
    "github.com/xaenox/memo-bot/internal/models"
)

type MemoryStorage struct {
    mu       sync.RWMutex
    users    map[int64]*models.User
    messages map[string]*models.Message  
    threads  map[int64]*models.Thread
}

func NewMemoryStorage() *MemoryStorage {
    return &MemoryStorage{
        users:    make(map[int64]*models.User),
        messages: make(map[string]*models.Message),
        threads:  make(map[int64]*models.Thread),
    }
}

// User methods
func (s *MemoryStorage) GetUser(ctx context.Context, id int64) (*models.User, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if user, exists := s.users[id]; exists {
        return user, nil
    }
    return &models.User{
        ID:         id,
        LastUsedAt: time.Now(),
    }, nil
}

func (s *MemoryStorage) UpdateUser(ctx context.Context, user *models.User) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    user.LastUsedAt = time.Now()
    s.users[user.ID] = user
    return nil
}

func (s *MemoryStorage) AddCategory(ctx context.Context, userID int64, category string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    user, exists := s.users[userID]
    if !exists {
        user = &models.User{
            ID:         userID,
            Categories: []string{},
            LastUsedAt: time.Now(),
        }
    }

    // Check if category already exists
    for _, c := range user.Categories {
        if c == category {
            return nil
        }
    }

    user.Categories = append(user.Categories, category)
    user.LastUsedAt = time.Now()
    s.users[userID] = user
    return nil
}

func (s *MemoryStorage) AddTag(ctx context.Context, userID int64, tag string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    user, exists := s.users[userID]
    if !exists {
        user = &models.User{
            ID:         userID,
            Tags:       []string{},
            LastUsedAt: time.Now(),
        }
    }

    // Check if tag already exists
    for _, t := range user.Tags {
        if t == tag {
            return nil
        }
    }

    user.Tags = append(user.Tags, tag)
    user.LastUsedAt = time.Now()
    s.users[userID] = user
    return nil
}

func (s *MemoryStorage) GetUserCategories(ctx context.Context, userID int64) ([]string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if user, exists := s.users[userID]; exists {
        return user.Categories, nil
    }
    return []string{}, nil
}

func (s *MemoryStorage) GetUserTags(ctx context.Context, userID int64) ([]string, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    if user, exists := s.users[userID]; exists {
        return user.Tags, nil
    }
    return []string{}, nil
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
