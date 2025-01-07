package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"github.com/xaenox/memo-bot/internal/models"
	"os"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(host string, port int, user, password, dbname, sslmode string) (*PostgresStorage, error) {
	var db *sql.DB
	var err error

	// If host is empty, assume we're using a connection URL from environment variable
	if host == "" {
		// Get DATABASE_URL from environment
		db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	} else {
		// Use individual connection parameters
		connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
		db, err = sql.Open("postgres", connStr)
	}

	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

func (s *PostgresStorage) CreateNote(ctx context.Context, note *models.Note) error {
	query := `
		INSERT INTO notes (user_id, content, type, tags, file_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, created_at, updated_at`
	
	return s.db.QueryRowContext(ctx, query,
		note.UserID,
		note.Content,
		note.Type,
		pq.Array(note.Tags),
		note.FileID,
	).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)
}

func (s *PostgresStorage) GetNotesByUserID(ctx context.Context, userID int64) ([]models.Note, error) {
	query := `
		SELECT id, user_id, content, type, tags, file_id, created_at, updated_at
		FROM notes
		WHERE user_id = $1
		ORDER BY created_at DESC`
	
	return s.queryNotes(ctx, query, userID)
}

func (s *PostgresStorage) GetNotesByTag(ctx context.Context, userID int64, tag string) ([]models.Note, error) {
	query := `
		SELECT id, user_id, content, type, tags, file_id, created_at, updated_at
		FROM notes
		WHERE user_id = $1 AND $2 = ANY(tags)
		ORDER BY created_at DESC`
	
	return s.queryNotes(ctx, query, userID, tag)
}

func (s *PostgresStorage) UpdateNoteTags(ctx context.Context, noteID int64, tags []string) error {
	query := `
		UPDATE notes
		SET tags = $1, updated_at = NOW()
		WHERE id = $2`
	
	result, err := s.db.ExecContext(ctx, query, pq.Array(tags), noteID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("note with id %d not found", noteID)
	}

	return nil
}

func (s *PostgresStorage) queryNotes(ctx context.Context, query string, args ...interface{}) ([]models.Note, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []models.Note
	for rows.Next() {
		var note models.Note
		var tags []string
		if err := rows.Scan(
			&note.ID,
			&note.UserID,
			&note.Content,
			&note.Type,
			pq.Array(&tags),
			&note.FileID,
			&note.CreatedAt,
			&note.UpdatedAt,
		); err != nil {
			return nil, err
		}
		note.Tags = tags
		notes = append(notes, note)
	}

	return notes, rows.Err()
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
} 