package storage

import (
	"context"
	"fmt"
	"sort"
	"github.com/xaenox/memo-bot/internal/models"
	"sync"
	"time"
)

type threadInfo struct {
	ID         string
	UserID     int64
	CreatedAt  time.Time
	LastUsedAt time.Time
}

type MemoryStorage struct {
	mu       sync.RWMutex
	users    map[int64]*models.User
	messages map[string]*models.Message
	threads  map[int64]threadInfo
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		users:    make(map[int64]*models.User),
		messages: make(map[string]*models.Message),
		threads:  make(map[int64]threadInfo),
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

func (s *MemoryStorage) GetThread(ctx context.Context, userID int64) (*models.Thread, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if thread, exists := s.threads[userID]; exists {
		return &models.Thread{
			ID:         thread.ID,
			UserID:     thread.UserID,
			CreatedAt:  thread.CreatedAt,
			LastUsedAt: thread.LastUsedAt,
		}, nil
	}
	return nil, nil
}

func (s *MemoryStorage) SaveThread(ctx context.Context, thread *models.Thread) error {
	if thread == nil {
		return fmt.Errorf("thread cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.threads[thread.UserID] = threadInfo{
		ID:         thread.ID,
		UserID:     thread.UserID,
		CreatedAt:  thread.CreatedAt,
		LastUsedAt: thread.LastUsedAt,
	}
	return nil
}

func (s *MemoryStorage) UpdateThreadLastUsed(ctx context.Context, userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if thread, exists := s.threads[userID]; exists {
		thread.LastUsedAt = time.Now()
		s.threads[userID] = thread
		return nil
	}
	return ErrNotFound
}

func (s *MemoryStorage) DeleteThread(userID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.threads, userID)
	return nil
}

func (s *MemoryStorage) DeleteMessage(ctx context.Context, id string) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.messages[id]; !exists {
        return ErrNotFound
    }

    delete(s.messages, id)
    return nil
}

func (s *MemoryStorage) SaveMessage(ctx context.Context, msg *models.Message) error {
    if msg == nil {
        return fmt.Errorf("message cannot be nil")
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    s.messages[msg.ID] = msg
    return nil
}

func (s *MemoryStorage) GetUserMessages(ctx context.Context, userID int64, limit int, offset int) ([]*models.Message, error) {
    s.mu.RLock()
    defer s.mu.RUnlock()

    var messages []*models.Message
    for _, msg := range s.messages {
        if msg.UserID == userID {
            messages = append(messages, msg)
        }
    }

    // Sort messages by creation time (newest first)
    sort.Slice(messages, func(i, j int) bool {
        return messages[i].CreatedAt.After(messages[j].CreatedAt)
    })

    // Apply limit and offset
    if offset >= len(messages) {
        return []*models.Message{}, nil
    }

    end := offset + limit
    if end > len(messages) {
        end = len(messages)
    }

    return messages[offset:end], nil
}
