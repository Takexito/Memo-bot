package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/xaenox/memo-bot/internal/models"
	"time"
)

//go:embed migrations.sql
var migrations embed.FS

type DatabaseConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	SSLMode     string
	UseInMemory bool
}

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(config DatabaseConfig) (*PostgresStorage, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}

	storage := &PostgresStorage{db: db}

	// Initialize database schema
	if err := storage.initializeSchema(); err != nil {
		return nil, fmt.Errorf("error initializing database schema: %v", err)
	}

	return storage, nil
}

func (s *PostgresStorage) initializeSchema() error {
	// Read migrations file
	migrationSQL, err := migrations.ReadFile("migrations.sql")
	if err != nil {
		return fmt.Errorf("error reading migrations file: %v", err)
	}

	// Execute migrations
	_, err = s.db.Exec(string(migrationSQL))
	if err != nil {
		return fmt.Errorf("error executing migrations: %v", err)
	}

	return nil
}

func (p *PostgresStorage) CreateNote(note *models.Note) error {
	query := `
        INSERT INTO notes (user_id, content, type, tags, file_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
        RETURNING id`

	return p.db.QueryRow(
		query,
		note.UserID,
		note.Content,
		note.Type,
		pq.Array(note.Tags), // Wrap tags with pq.Array
		note.FileID,
	).Scan(&note.ID)
}

func (p *PostgresStorage) GetNotesByUserID(userID int64) ([]*models.Note, error) {
	query := `SELECT id, user_id, content, type, tags, file_id, created_at, updated_at 
              FROM notes WHERE user_id = $1 
              ORDER BY created_at DESC LIMIT 10`

	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		note := &models.Note{}
		err := rows.Scan(
			&note.ID,
			&note.UserID,
			&note.Content,
			&note.Type,
			pq.Array(&note.Tags), // Wrap tags with pq.Array
			&note.FileID,
			&note.CreatedAt,
			&note.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (p *PostgresStorage) GetNotesByTag(userID int64, tag string) ([]*models.Note, error) {
	query := `SELECT id, user_id, content, type, tags, file_id, created_at, updated_at 
              FROM notes 
              WHERE user_id = $1 AND $2 = ANY(tags)
              ORDER BY created_at DESC LIMIT 10`

	rows, err := p.db.Query(query, userID, tag)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notes []*models.Note
	for rows.Next() {
		note := &models.Note{}
		err := rows.Scan(
			&note.ID,
			&note.UserID,
			&note.Content,
			&note.Type,
			pq.Array(&note.Tags), // Wrap tags with pq.Array
			&note.FileID,
			&note.CreatedAt,
			&note.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	return notes, rows.Err()
}

func (s *PostgresStorage) UpdateNoteTags(noteID int64, tags []string) error {
	query := `
		UPDATE notes
		SET tags = $1, updated_at = $2
		WHERE id = $3`

	result, err := s.db.Exec(query, tags, time.Now(), noteID)
	if err != nil {
		return fmt.Errorf("error updating note tags: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("note not found")
	}

	return nil
}

func (s *PostgresStorage) Close() error {
	return s.db.Close()
}
