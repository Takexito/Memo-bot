package models

import "time"

// User represents a bot user and their metadata
type User struct {
	ID         int64     `json:"id"`
	ThreadID   string    `json:"thread_id,omitempty"`
	Categories []string  `json:"categories"`
	Tags       []string  `json:"tags"`
	LastUsedAt time.Time `json:"last_used_at"`
}

// Message represents a classified message
type Message struct {
	ID        string    `json:"id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	Category  string    `json:"category"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
}

// Thread represents an AI assistant conversation thread
type Thread struct {
	ID         string    `json:"id"`
	UserID     int64     `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsedAt time.Time `json:"last_used_at"`
}
