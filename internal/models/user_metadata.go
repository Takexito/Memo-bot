package models

import "time"

type UserMetadata struct {
	UserID     int64     `json:"user_id"`
	ThreadID   string    `json:"thread_id"`
	Categories []string  `json:"categories"`
	Tags       []string  `json:"tags"`
	LastUsedAt time.Time `json:"last_used_at"`
}
