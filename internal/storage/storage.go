package storage

import (
	"github.com/xaenox/memo-bot/internal/models"
)

type Storage interface {
	CreateNote(note *models.Note) error
	GetNotesByUserID(userID int64) ([]*models.Note, error)
	GetNotesByTag(userID int64, tag string) ([]*models.Note, error)
	UpdateNoteTags(noteID int64, tags []string) error
	Close() error
}

type ThreadStorage interface {
	GetThread(userID int64) (string, error)
	SaveThread(userID int64, threadID string) error
	UpdateThreadLastUsed(userID int64) error
	DeleteThread(userID int64) error
}
