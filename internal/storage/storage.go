package storage

import "github.com/xaenox/memo-bot/internal/models"

type Storage interface {
	GetUserMetadata(userID int64) (*models.UserMetadata, error)
	UpdateUserMetadata(metadata *models.UserMetadata) error
	AddUserCategory(userID int64, category string) error
	AddUserTag(userID int64, tag string) error
	Close() error

	// Embed ThreadStorage interface
	ThreadStorage
}

type ThreadStorage interface {
	GetThread(userID int64) (string, error)
	SaveThread(userID int64, threadID string) error
	UpdateThreadLastUsed(userID int64) error
	DeleteThread(userID int64) error
}
