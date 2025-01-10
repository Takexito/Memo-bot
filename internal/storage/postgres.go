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

func (p *PostgresStorage) GetMessageByID(ctx context.Context, id string) (*models.Message, error) {
    query := `
        SELECT id, user_id, content, category, tags, created_at
        FROM messages
        WHERE id = $1`

    var msg models.Message
    err := p.db.QueryRowContext(ctx, query, id).Scan(
        &msg.ID,
        &msg.UserID,
        &msg.Content,
        &msg.Category,
        pq.Array(&msg.Tags),
        &msg.CreatedAt,
    )

    if err == sql.ErrNoRows {
        return nil, ErrNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get message: %w", err)
    }
    return &msg, nil
}

func (p *PostgresStorage) DeleteMessage(ctx context.Context, id string) error {
    result, err := p.db.ExecContext(ctx, `
        DELETE FROM messages 
        WHERE id = $1`,
        id,
    )
    if err != nil {
        return fmt.Errorf("failed to delete message: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get affected rows: %w", err)
    }
    if rows == 0 {
        return ErrNotFound
    }
    return nil
}

func (p *PostgresStorage) AddCategory(ctx context.Context, userID int64, category string) error {
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

    _, err := p.db.ExecContext(ctx, query, userID, category)
    if err != nil {
        return fmt.Errorf("failed to add category: %w", err)
    }
    return nil
}

func (p *PostgresStorage) AddTag(ctx context.Context, userID int64, tag string) error {
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

    _, err := p.db.ExecContext(ctx, query, userID, tag)
    if err != nil {
        return fmt.Errorf("failed to add tag: %w", err)
    }
    return nil
}

func (p *PostgresStorage) GetUserCategories(ctx context.Context, userID int64) ([]string, error) {
    query := `
        SELECT categories
        FROM user_metadata
        WHERE user_id = $1`

    var categories []string
    err := p.db.QueryRowContext(ctx, query, userID).Scan(pq.Array(&categories))
    if err == sql.ErrNoRows {
        return []string{}, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user categories: %w", err)
    }
    return categories, nil
}

func (p *PostgresStorage) GetUserTags(ctx context.Context, userID int64) ([]string, error) {
    query := `
        SELECT tags
        FROM user_metadata
        WHERE user_id = $1`

    var tags []string
    err := p.db.QueryRowContext(ctx, query, userID).Scan(pq.Array(&tags))
    if err == sql.ErrNoRows {
        return []string{}, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get user tags: %w", err)
    }
    return tags, nil
}

func (p *PostgresStorage) GetThread(ctx context.Context, userID int64) (*models.Thread, error) {
    query := `
        SELECT id, user_id, created_at, last_used_at
        FROM threads
        WHERE user_id = $1`

    var thread models.Thread
    err := p.db.QueryRowContext(ctx, query, userID).Scan(
        &thread.ID,
        &thread.UserID,
        &thread.CreatedAt,
        &thread.LastUsedAt,
    )

    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get thread: %w", err)
    }
    return &thread, nil
}

func (p *PostgresStorage) SaveThread(ctx context.Context, thread *models.Thread) error {
    query := `
        INSERT INTO threads (id, user_id, created_at, last_used_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (user_id) DO UPDATE SET
            id = EXCLUDED.id,
            created_at = EXCLUDED.created_at,
            last_used_at = EXCLUDED.last_used_at`

    _, err := p.db.ExecContext(ctx, query,
        thread.ID,
        thread.UserID,
        thread.CreatedAt,
        thread.LastUsedAt,
    )
    if err != nil {
        return fmt.Errorf("failed to save thread: %w", err)
    }
    return nil
}

func (p *PostgresStorage) UpdateThreadLastUsed(ctx context.Context, userID int64) error {
    query := `
        UPDATE threads
        SET last_used_at = NOW()
        WHERE user_id = $1`

    result, err := p.db.ExecContext(ctx, query, userID)
    if err != nil {
        return fmt.Errorf("failed to update thread last used: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get affected rows: %w", err)
    }
    if rows == 0 {
        return ErrNotFound
    }
    return nil
}

func (p *PostgresStorage) DeleteThread(ctx context.Context, userID int64) error {
    result, err := p.db.ExecContext(ctx, `
        DELETE FROM threads 
        WHERE user_id = $1`,
        userID,
    )
    if err != nil {
        return fmt.Errorf("failed to delete thread: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get affected rows: %w", err)
    }
    if rows == 0 {
        return ErrNotFound
    }
    return nil
}
