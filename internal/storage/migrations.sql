-- Drop existing objects
DROP TABLE IF EXISTS notes CASCADE;
DROP TYPE IF EXISTS content_type CASCADE;

-- Add user metadata table
CREATE TABLE IF NOT EXISTS user_metadata (
    user_id BIGINT PRIMARY KEY,
    thread_id VARCHAR(255),
    categories TEXT[] DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    last_used_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_user_metadata_last_used ON user_metadata(last_used_at);
