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

func (p *PostgresStorage) GetUserMetadata(userID int64) (*UserMetadata, error) {
	query := `
		SELECT user_id, thread_id, categories, tags, last_used_at
		FROM user_metadata
		WHERE user_id = $1`

	metadata := &UserMetadata{UserID: userID}
	err := p.db.QueryRow(query, userID).Scan(
		&metadata.UserID,
		&metadata.ThreadID,
		pq.Array(&metadata.Categories),
		pq.Array(&metadata.Tags),
		&metadata.LastUsedAt,
	)

	if err == sql.ErrNoRows {
		// Initialize new metadata if not exists
		metadata.LastUsedAt = time.Now()
		return metadata, nil
	}
	return metadata, err
}

func (p *PostgresStorage) UpdateUserMetadata(metadata *UserMetadata) error {
	query := `
		INSERT INTO user_metadata (user_id, thread_id, categories, tags, last_used_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			thread_id = EXCLUDED.thread_id,
			categories = EXCLUDED.categories,
			tags = EXCLUDED.tags,
			last_used_at = EXCLUDED.last_used_at`

	_, err := p.db.Exec(query,
		metadata.UserID,
		metadata.ThreadID,
		pq.Array(metadata.Categories),
		pq.Array(metadata.Tags),
		time.Now(),
	)
	return err
}

func (p *PostgresStorage) AddUserCategory(userID int64, category string) error {
	query := `
		INSERT INTO user_metadata (user_id, categories, last_used_at)
		VALUES ($1, ARRAY[$2], NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			categories = array_append(
				array_remove(user_metadata.categories, $2),
				$2
			),
			last_used_at = NOW()
		WHERE NOT ($2 = ANY(user_metadata.categories))`

	_, err := p.db.Exec(query, userID, category)
	return err
}

func (p *PostgresStorage) AddUserTag(userID int64, tag string) error {
	query := `
		INSERT INTO user_metadata (user_id, tags, last_used_at)
		VALUES ($1, ARRAY[$2], NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			tags = array_append(
				array_remove(user_metadata.tags, $2),
				$2
			),
			last_used_at = NOW()
		WHERE NOT ($2 = ANY(user_metadata.tags))`

	_, err := p.db.Exec(query, userID, tag)
	return err
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

func (p *PostgresStorage) GetThread(userID int64) (string, error) {
	var threadID string
	err := p.db.QueryRow(`
		SELECT thread_id 
		FROM assistant_threads 
		WHERE user_id = $1`, userID).Scan(&threadID)
	
	if err == sql.ErrNoRows {
		return "", nil
	}
	return threadID, err
}

func (p *PostgresStorage) SaveThread(userID int64, threadID string) error {
	_, err := p.db.Exec(`
		INSERT INTO assistant_threads (user_id, thread_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			thread_id = EXCLUDED.thread_id,
			last_used_at = CURRENT_TIMESTAMP`,
		userID, threadID)
	return err
}

func (p *PostgresStorage) UpdateThreadLastUsed(userID int64) error {
	_, err := p.db.Exec(`
		UPDATE assistant_threads 
		SET last_used_at = CURRENT_TIMESTAMP
		WHERE user_id = $1`,
		userID)
	return err
}

func (p *PostgresStorage) DeleteThread(userID int64) error {
	_, err := p.db.Exec(`
		DELETE FROM assistant_threads 
		WHERE user_id = $1`,
		userID)
	return err
}
