package service

import (
    "context"
    "encoding/json"
    "net/http"
    "regexp"
    "sync"
    "time"
    "github.com/Prototype-1/TargetService/config"
    "github.com/Prototype-1/TargetService/internal/model"
    "github.com/Prototype-1/TargetService/internal/repository"
    "go.uber.org/zap"
)

type SyncService struct {
    Repo    *repository.UserRepository
    Logger  *zap.Logger
    Config  config.Config
    Client  *http.Client
    EmailRe *regexp.Regexp
}

func NewSyncService(repo *repository.UserRepository, logger *zap.Logger, cfg config.Config) *SyncService {
    return &SyncService{
        Repo:    repo,
        Logger:  logger,
        Config:  cfg,
        Client:  &http.Client{Timeout: 5 * time.Second},
        EmailRe: regexp.MustCompile(`^[a-zA-Z0-9._%%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`),
    }
}

func (s *SyncService) Start(ctx context.Context) {
    ticker := time.NewTicker(time.Duration(s.Config.SyncInterval) * time.Second)
    defer ticker.Stop()

    s.Logger.Info("SyncService started", zap.Int("interval", s.Config.SyncInterval))

    for {
        select {
        case <-ticker.C:
            s.Logger.Info("Fetching new user profiles from SourceService")
            s.syncBatch(ctx)
        case <-ctx.Done():
            s.Logger.Info("SyncService shutting down gracefully")
            return
        }
    }
}

func (s *SyncService) syncBatch(ctx context.Context) {
    req, err := http.NewRequestWithContext(ctx, "GET", s.Config.SourceURL, nil)
    if err != nil {
        s.Logger.Error("creating request failed", zap.Error(err))
        return
    }

    resp, err := s.Client.Do(req)
    if err != nil {
        s.Logger.Error("fetch from SourceService failed", zap.Error(err))
        return
    }
    defer resp.Body.Close()

    var users []model.UserProfile
    if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
        s.Logger.Error("JSON decode failed", zap.Error(err))
        return
    }

	//zap for structured logging
    s.Logger.Info("Fetched user profiles", zap.Int("count", len(users)))

    // Worker pool whuch limits max concurrent DB writes
	//WaitGroup to wait for all processing
    var wg sync.WaitGroup
    concurrency := 5
    sem := make(chan struct{}, concurrency)

    for _, user := range users {
        wg.Add(1)
	//structs are more flexible that this part block if concurrency is full
        sem <- struct{}{} 
        go func(u model.UserProfile) {
            defer wg.Done()
            defer func() { <-sem }()

            s.processUser(ctx, u)
        }(user)
    }

    wg.Wait()
}

func (s *SyncService) processUser(ctx context.Context, user model.UserProfile) {
    // This specific part validates the email
    if !s.EmailRe.MatchString(user.Email) {
        user.SyncStatus = "failed_validation"
        user.SyncMessage = "Invalid email format"
        s.Logger.Warn("Validation failed", zap.String("id", user.ID), zap.String("reason", user.SyncMessage))
        s.Repo.InsertOrUpdateUser(ctx, user)
        return
    }

    //  This specific part validates the status
    if user.Status != "active" && user.Status != "pending" {
        user.SyncStatus = "skipped"
        user.SyncMessage = "Status is neither active nor pending"
        s.Logger.Warn("Skipped user", zap.String("id", user.ID), zap.String("reason", user.SyncMessage))
        s.Repo.InsertOrUpdateUser(ctx, user)
        return
    }

    //  Here we check for  existing user/s
    existing, _ := s.Repo.GetUserByID(ctx, user.ID)
    if existing != nil && existing.LastUpdatedAt >= user.LastUpdatedAt {
        user.SyncStatus = "skipped"
        user.SyncMessage = "Not newer than existing record"
        s.Logger.Info("Skipped older record", zap.String("id", user.ID))
        s.Repo.InsertOrUpdateUser(ctx, user)
        return
    }

    // Once we pass all the checks, we will mark ot as synced
    user.SyncStatus = "synced"
    user.SyncMessage = "Successfully synced"
    if err := s.Repo.InsertOrUpdateUser(ctx, user); err != nil {
        s.Logger.Error("DB insert/update failed", zap.Error(err), zap.String("id", user.ID))
    } else {
        s.Logger.Info("User synced", zap.String("id", user.ID))
    }
}
