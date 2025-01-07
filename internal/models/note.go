package models

import (
	"time"
)

type ContentType string

const (
	TextContent    ContentType = "text"
	ImageContent   ContentType = "image"
	VideoContent   ContentType = "video"
	LinkContent    ContentType = "link"
	DocumentContent ContentType = "document"
)

type Note struct {
	ID        int64       `json:"id"`
	UserID    int64       `json:"user_id"`
	Content   string      `json:"content"`
	Type      ContentType `json:"type"`
	Tags      []string    `json:"tags"`
	FileID    string      `json:"file_id,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
} 