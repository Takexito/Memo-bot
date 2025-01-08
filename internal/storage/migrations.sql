-- Create notes table
CREATE TABLE IF NOT EXISTS notes (
    id SERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    content TEXT NOT NULL,
    content_type VARCHAR(50) NOT NULL,
    file_id VARCHAR(255),
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for faster user_id lookups
CREATE INDEX IF NOT EXISTS idx_notes_user_id ON notes(user_id);

-- Create index for faster tag searches
CREATE INDEX IF NOT EXISTS idx_notes_tags ON notes USING GIN(tags); 