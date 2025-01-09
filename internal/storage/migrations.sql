-- Drop existing objects
DROP TABLE IF EXISTS notes;
DROP TYPE IF EXISTS content_type CASCADE;

-- Create fresh schema
CREATE TYPE content_type AS ENUM (
    'text',
    'image',
    'video',
    'link',
    'document'
);

CREATE TABLE notes (
                       id SERIAL PRIMARY KEY,
                       user_id BIGINT NOT NULL,
                       content TEXT NOT NULL,
                       type content_type NOT NULL,
                       file_id VARCHAR(255),
                       tags TEXT[] DEFAULT '{}',
                       created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                       updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_notes_user_id ON notes(user_id);
CREATE INDEX idx_notes_tags ON notes USING GIN(tags);