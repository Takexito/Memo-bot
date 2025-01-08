package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/xaenox/memo-bot/internal/models"
)

type MemoryStorage struct {
	mu     sync.RWMutex
	notes  map[int64]models.Note
	lastID int64
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		notes:  make(map[int64]models.Note),
		lastID: 0,
	}
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

func (s *MemoryStorage) GetNotesByUserID(userID int64) ([]models.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.Note
	for _, note := range s.notes {
		if note.UserID == userID {
			result = append(result, note)
		}
	}

	// Sort by created_at DESC (newest first)
	sortNotesByCreatedAt(result)
	return result, nil
}

func (s *MemoryStorage) GetNotesByTag(userID int64, tag string) ([]models.Note, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []models.Note
	for _, note := range s.notes {
		if note.UserID == userID {
			for _, t := range note.Tags {
				if t == tag {
					result = append(result, note)
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

// Helper function to sort notes by created_at in descending order
func sortNotesByCreatedAt(notes []models.Note) {
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
