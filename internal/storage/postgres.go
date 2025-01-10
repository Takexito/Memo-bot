package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/lib/pq"
	"github.com/xaenox/memo-bot/internal/models"
	"go.uber.org/zap"
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
	db     *sql.DB
	logger *zap.Logger
}

// User-related methods
func (p *PostgresStorage) GetUser(ctx context.Context, id int64) (*models.User, error) {
	query := `
		SELECT user_id, thread_id, categories, tags, last_used_at
		FROM user_metadata
		WHERE user_id = $1`

	user := &models.User{ID: id}
	err := p.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.ThreadID,
		pq.Array(&user.Categories),
		pq.Array(&user.Tags),
		&user.LastUsedAt,
	)

	if err == sql.ErrNoRows {
		return &models.User{
			ID:         id,
			LastUsedAt: time.Now(),
		}, nil
	}
	return user, err
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

func NewPostgresStorage(config DatabaseConfig, logger *zap.Logger) (*PostgresStorage, error) {
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

	storage := &PostgresStorage{
		db: db,
		logger: logger,
	}

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
