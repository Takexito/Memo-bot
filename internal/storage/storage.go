package storage

import (
	"context"
	"github.com/xaenox/memo-bot/internal/models"
)

type Storage interface {
	CreateNote(ctx context.Context, note *models.Note) error
	GetNotesByUserID(ctx context.Context, userID int64) ([]models.Note, error)
	GetNotesByTag(ctx context.Context, userID int64, tag string) ([]models.Note, error)
	UpdateNoteTags(ctx context.Context, noteID int64, tags []string) error
	Close() error
} 