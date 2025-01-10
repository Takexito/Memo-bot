-- Drop existing objects
DROP TABLE IF EXISTS notes CASCADE;
DROP TYPE IF EXISTS content_type CASCADE;

-- Add user metadata table
CREATE TABLE IF NOT EXISTS user_metadata (
    user_id BIGINT PRIMARY KEY,
    thread_id VARCHAR(255),
    categories TEXT[] DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    max_tags INTEGER DEFAULT 5,
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create threads table
CREATE TABLE IF NOT EXISTS threads (
    id VARCHAR(255) PRIMARY KEY,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES user_metadata(user_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_metadata_last_used ON user_metadata(last_used_at);
CREATE INDEX IF NOT EXISTS idx_threads_user_id ON threads(user_id);
