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
)

// Storage combines all storage interfaces
type Storage interface {
    UserStorage
    MessageStorage
    ThreadStorage
    Close() error
}

// UserStorage handles user-related operations
type UserStorage interface {
    GetUser(ctx context.Context, id int64) (*models.User, error)
    UpdateUser(ctx context.Context, user *models.User) error
    AddCategory(ctx context.Context, userID int64, category string) error
    AddTag(ctx context.Context, userID int64, tag string) error
    GetUserCategories(ctx context.Context, userID int64) ([]string, error)
    GetUserTags(ctx context.Context, userID int64) ([]string, error)
}

// MessageStorage handles message operations
type MessageStorage interface {
    SaveMessage(ctx context.Context, msg *models.Message) error
    GetUserMessages(ctx context.Context, userID int64, limit int, offset int) ([]*models.Message, error)
    GetMessageByID(ctx context.Context, id string) (*models.Message, error)
    DeleteMessage(ctx context.Context, id string) error
}

// ThreadStorage handles AI assistant thread operations
type ThreadStorage interface {
    GetThread(ctx context.Context, userID int64) (*models.Thread, error)
    SaveThread(ctx context.Context, thread *models.Thread) error
    UpdateThreadLastUsed(ctx context.Context, userID int64) error
    DeleteThread(ctx context.Context, userID int64) error
}
