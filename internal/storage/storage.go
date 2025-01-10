package storage

import (
	"context"
	"errors"
	"github.com/xaenox/memo-bot/internal/models"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrAlreadyExists = errors.New("record already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrDatabase      = errors.New("database error")
	ErrDuplicate     = errors.New("duplicate record")
	ErrConnection    = errors.New("database connection error")
	ErrTransaction   = errors.New("transaction error")
	ErrConstraint    = errors.New("constraint violation")
)

// IsDatabaseError checks if an error is a database-related error
func IsDatabaseError(err error) bool {
	return errors.Is(err, ErrDatabase) ||
		errors.Is(err, ErrConnection) ||
		errors.Is(err, ErrTransaction) ||
		errors.Is(err, ErrConstraint)
}

// Storage combines all storage interfaces
type Storage interface {
	UserStorage
	ThreadStorage
	Close() error
}

// UserStorage handles user-related operations
type UserStorage interface {
	GetUser(ctx context.Context, id int64) (*models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	AddCategory(ctx context.Context, userID int64, category string) error
	RemoveCategory(ctx context.Context, userID int64, category string) error
	UpdateUserMaxTags(ctx context.Context, userID int64, maxTags int) error
	AddTag(ctx context.Context, userID int64, tag string) error
	GetUserCategories(ctx context.Context, userID int64) ([]string, error)
	GetUserTags(ctx context.Context, userID int64) ([]string, error)
}

// ThreadStorage handles AI assistant thread operations
type ThreadStorage interface {
	GetThread(ctx context.Context, userID int64) (*models.Thread, error)
	SaveThread(ctx context.Context, thread *models.Thread) error
	UpdateThreadLastUsed(ctx context.Context, userID int64) error
	DeleteThread(ctx context.Context, userID int64) error
}
