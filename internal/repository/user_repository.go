package repository

import (
    "context"
    "fmt"
    "time"

    "github.com/Prototype-1/TargetService/internal/model"
    "github.com/jackc/pgx/v4/pgxpool"
    "go.uber.org/zap"
)

type UserRepository struct {
    DB     *pgxpool.Pool
    Logger *zap.Logger
}

func NewUserRepository(dbURL string, logger *zap.Logger) (*UserRepository, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    pool, err := pgxpool.Connect(ctx, dbURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    return &UserRepository{
        DB:     pool,
        Logger: logger,
    }, nil
}

func (r *UserRepository) Close() {
    r.DB.Close()
}

// GetUserByID returns the existing user by ID, or nil if not found
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*model.UserProfile, error) {
    var user model.UserProfile

    query := `
        SELECT id, name, email, mobile, status, last_updated_at, sync_status, sync_message
        FROM users WHERE id = $1
    `

    err := r.DB.QueryRow(ctx, query, id).Scan(
        &user.ID,
        &user.Name,
        &user.Email,
        &user.Mobile,
        &user.Status,
        &user.LastUpdatedAt,
        &user.SyncStatus,
        &user.SyncMessage,
    )

    if err != nil {
        return nil, nil 
    }

    return &user, nil
}

func (r *UserRepository) InsertOrUpdateUser(ctx context.Context, user model.UserProfile) error {
    query := `
    INSERT INTO users (id, name, email, mobile, status, last_updated_at, sync_status, sync_message)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    ON CONFLICT (id)
    DO UPDATE SET 
        name = EXCLUDED.name,
        email = EXCLUDED.email,
        mobile = EXCLUDED.mobile,
        status = EXCLUDED.status,
        last_updated_at = EXCLUDED.last_updated_at,
        sync_status = EXCLUDED.sync_status,
        sync_message = EXCLUDED.sync_message
    `

    _, err := r.DB.Exec(ctx, query,
        user.ID,
        user.Name,
        user.Email,
        user.Mobile,
        user.Status,
        user.LastUpdatedAt,
        user.SyncStatus,
        user.SyncMessage,
    )

    if err != nil {
        r.Logger.Error("InsertOrUpdateUser failed", zap.Error(err))
    }

    return err
}
