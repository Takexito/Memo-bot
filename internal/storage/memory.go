package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/xaenox/memo-bot/internal/models"
)

type threadInfo struct {
	ThreadID    string
	CreatedAt   time.Time
	LastUsedAt  time.Time
}

type MemoryStorage struct {
	mu          sync.RWMutex
	notes       map[int64]models.Note
	lastID      int64
	userMeta    map[int64]*UserMetadata
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		notes:    make(map[int64]models.Note),
		lastID:   0,
		userMeta: make(map[int64]*UserMetadata),
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

func (s *MemoryStorage) CreateNote(note *models.Note) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastID++
	note.ID = s.lastID
	note.CreatedAt = time.Now()
	note.UpdatedAt = note.CreatedAt

	s.notes[note.ID] = *note
	return nil
}

func (s *MemoryStorage) GetNotesByUserID(userID int64) ([]*models.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Note
	for _, note := range s.notes {
		if note.UserID == userID {
			result = append(result, &note)
		}
	}

	// Sort by created_at DESC (newest first)
	sortNotesByCreatedAt(result)
	return result, nil
}

func (s *MemoryStorage) GetNotesByTag(userID int64, tag string) ([]*models.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*models.Note
	for _, note := range s.notes {
		if note.UserID == userID {
			for _, t := range note.Tags {
				if t == tag {
					result = append(result, &note)
					break
				}
			}
		}
	}

	// Sort by created_at DESC (newest first)
	sortNotesByCreatedAt(result)
	return result, nil
}

func (s *MemoryStorage) UpdateNoteTags(noteID int64, tags []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, exists := s.notes[noteID]
	if !exists {
		return fmt.Errorf("note with id %d not found", noteID)
	}

	note.Tags = tags
	note.UpdatedAt = time.Now()
	s.notes[noteID] = note
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
		ThreadID:    threadID,
		CreatedAt:   time.Now(),
		LastUsedAt:  time.Now(),
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

// Helper function to sort notes by created_at in descending order
func sortNotesByCreatedAt(notes []*models.Note) {
	if len(notes) <= 1 {
		return
	}

	for i := 0; i < len(notes)-1; i++ {
		for j := i + 1; j < len(notes); j++ {
			if notes[i].CreatedAt.Before(notes[j].CreatedAt) {
				notes[i], notes[j] = notes[j], notes[i]
			}
		}
	}
}
