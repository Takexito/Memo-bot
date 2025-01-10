package storage

import (
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
    // User metadata methods
    GetUserMetadata(userID int64) (*UserMetadata, error)
    UpdateUserMetadata(metadata *UserMetadata) error
    AddUserCategory(userID int64, category string) error
    AddUserTag(userID int64, tag string) error
    Close() error
}
