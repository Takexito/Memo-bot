package models

import "time"

// Message represents a user message with its classification
type Message struct {
    ID        string    `json:"id"`
    UserID    int64     `json:"user_id"`
    Content   string    `json:"content"`
    Category  string    `json:"category"`
    Tags      []string  `json:"tags"`
    CreatedAt time.Time `json:"created_at"`
}

// User represents a bot user with their preferences and metadata
type User struct {
    ID         int64     `json:"id"`
    ThreadID   string    `json:"thread_id,omitempty"`
    Categories []string  `json:"categories"`
    Tags       []string  `json:"tags"`
    LastUsedAt time.Time `json:"last_used_at"`
}

// Classification represents the result of content analysis
type Classification struct {
    Category    string   `json:"category"`
    Tags        []string `json:"tags"`
    Summary     string   `json:"summary"`
    Confidence  float64  `json:"confidence"`
    RawResponse any      `json:"raw_response,omitempty"`
}

// Thread represents an AI assistant conversation thread
type Thread struct {
    ID         string    `json:"id"`
    UserID     int64     `json:"user_id"`
    CreatedAt  time.Time `json:"created_at"`
    LastUsedAt time.Time `json:"last_used_at"`
}
