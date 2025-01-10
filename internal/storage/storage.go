package storage

import (
	"github.com/xaenox/memo-bot/internal/models"
	"time"
)

type UserMetadata struct {
	UserID      int64     `json:"user_id"`
	ThreadID    string    `json:"thread_id"`
	Categories  []string  `json:"categories"`
	Tags        []string  `json:"tags"`
	LastUsedAt  time.Time `json:"last_used_at"`
}

type Storage interface {
	CreateNote(note *models.Note) error
	GetNotesByUserID(userID int64) ([]*models.Note, error)
	GetNotesByTag(userID int64, tag string) ([]*models.Note, error)
	UpdateNoteTags(noteID int64, tags []string) error
	Close() error
	
	// User metadata methods
	GetUserMetadata(userID int64) (*UserMetadata, error)
	UpdateUserMetadata(metadata *UserMetadata) error
	AddUserCategory(userID int64, category string) error
	AddUserTag(userID int64, tag string) error
}
