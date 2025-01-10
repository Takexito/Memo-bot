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

func (p *PostgresStorage) handleError(err error, operation string) error {
    if err == nil {
        return nil
    }

    // Log the error with context
    p.logger.Error("database error",
        zap.Error(err),
        zap.String("operation", operation))

    // Handle specific postgres errors
    if pqErr, ok := err.(*pq.Error); ok {
        switch pqErr.Code {
        case "23505": // unique_violation
            return fmt.Errorf("%w: %v", ErrDuplicate, err)
        case "23503": // foreign_key_violation
            return fmt.Errorf("%w: %v", ErrConstraint, err)
        case "23502": // not_null_violation
            return fmt.Errorf("%w: invalid input - %v", ErrInvalidInput, err)
        case "08000", "08003", "08006", "08001", "08004": // connection errors
            return fmt.Errorf("%w: %v", ErrConnection, err)
        }
        return fmt.Errorf("%w: %v", ErrDatabase, err)
    }

    // Handle other common errors
    switch {
    case err == sql.ErrNoRows:
        return ErrNotFound
    case err == sql.ErrTxDone:
        return fmt.Errorf("%w: transaction already closed", ErrTransaction)
    case err == sql.ErrConnDone:
        return fmt.Errorf("%w: connection already closed", ErrConnection)
    }

    // Generic database error
    return fmt.Errorf("%w: %v", ErrDatabase, err)
}

// User-related methods
func (p *PostgresStorage) GetUser(ctx context.Context, id int64) (*models.User, error) {
    if id == 0 {
        return nil, fmt.Errorf("%w: user_id cannot be zero", ErrInvalidInput)
    }

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
    
    if err != nil {
        return nil, p.handleError(err, "GetUser")
    }
    
    return user, nil
}

func (p *PostgresStorage) UpdateUser(ctx context.Context, user *models.User) error {
    // Input validation
    if user == nil {
        return fmt.Errorf("%w: user cannot be nil", ErrInvalidInput)
    }
    if user.ID == 0 {
        return fmt.Errorf("%w: user_id cannot be zero", ErrInvalidInput)
    }

    query := `
        INSERT INTO user_metadata (user_id, thread_id, categories, tags, last_used_at)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (user_id) DO UPDATE SET
            thread_id = EXCLUDED.thread_id,
            categories = EXCLUDED.categories,
            tags = EXCLUDED.tags,
            last_used_at = EXCLUDED.last_used_at`

    _, err := p.db.ExecContext(ctx, query,
        user.ID,
        user.ThreadID,
        pq.Array(user.Categories),
        pq.Array(user.Tags),
        user.LastUsedAt,
    )
    
    return p.handleError(err, "UpdateUser")
}



func (p *PostgresStorage) GetUserMessages(ctx context.Context, userID int64, limit int, offset int) ([]*models.Message, error) {
    query := `
        SELECT id, user_id, content, category, tags, created_at
        FROM messages
        WHERE user_id = $1
        ORDER BY created_at DESC
        LIMIT $2 OFFSET $3`

    rows, err := p.db.QueryContext(ctx, query, userID, limit, offset)
    if err != nil {
        return nil, p.handleError(err, "GetUserMessages")
    }
    defer rows.Close()

    var messages []*models.Message
    for rows.Next() {
        var msg models.Message
        err := rows.Scan(
            &msg.ID,
            &msg.UserID,
            &msg.Content,
            &msg.Category,
            pq.Array(&msg.Tags),
            &msg.CreatedAt,
        )
        if err != nil {
            return nil, p.handleError(err, "GetUserMessages")
        }
        messages = append(messages, &msg)
    }

    if err = rows.Err(); err != nil {
        return nil, p.handleError(err, "GetUserMessages")
    }

    return messages, nil
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

func (p *PostgresStorage) CheckHealth(ctx context.Context) error {
    // Check if connection is alive
    err := p.db.PingContext(ctx)
    if err != nil {
        return p.handleError(err, "CheckHealth")
    }

    // Optional: Check if we can perform a simple query
    _, err = p.db.ExecContext(ctx, "SELECT 1")
    if err != nil {
        return p.handleError(err, "CheckHealth")
    }

    return nil
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
