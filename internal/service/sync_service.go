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
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "go.uber.org/zap"
)

type SyncService struct {
    Repo    *repository.UserRepository
    Logger  *zap.Logger
    Config  config.Config
    Client  *http.Client
    EmailRe *regexp.Regexp
    ProfilesFetched prometheus.Counter
    ProfilesSynced  prometheus.Counter
    ProfilesSkipped prometheus.Counter
    SyncDuration    prometheus.Histogram
}

func NewSyncService(repo *repository.UserRepository, logger *zap.Logger, cfg config.Config) *SyncService {
    s := &SyncService{
        Repo:    repo,
        Logger:  logger,
        Config:  cfg,
        Client:  &http.Client{Timeout: 5 * time.Second},
        EmailRe: regexp.MustCompile(`^[a-zA-Z0-9._%%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`),

        ProfilesFetched: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "profiles_fetched_total",
            Help: "Total number of profiles fetched from SourceService",
        }),
        ProfilesSynced: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "profiles_synced_total",
            Help: "Total number of profiles successfully synced",
        }),
        ProfilesSkipped: prometheus.NewCounter(prometheus.CounterOpts{
            Name: "profiles_skipped_total",
            Help: "Total number of profiles skipped or failed validation",
        }),
        SyncDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "sync_batch_duration_seconds",
            Help:    "Duration of sync batch in seconds",
            Buckets: prometheus.DefBuckets,
        }),
    }

    //Here it register all metrics with Prometheus
    prometheus.MustRegister(s.ProfilesFetched, s.ProfilesSynced, s.ProfilesSkipped, s.SyncDuration)
    return s
}

func (s *SyncService) Start(ctx context.Context) {
    // Start the Prometheus metrics server
    go func() {
        s.Logger.Info("Metrics endpoint running on :2112/metrics")
        http.Handle("/metrics", promhttp.Handler())
        if err := http.ListenAndServe(":2112", nil); err != nil && err != http.ErrServerClosed {
            s.Logger.Fatal("Metrics HTTP server failed", zap.Error(err))
        }
    }()

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
    start := time.Now()

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

    s.ProfilesFetched.Add(float64(len(users)))
    s.Logger.Info("Fetched user profiles", zap.Int("count", len(users)))

    var wg sync.WaitGroup
    concurrency := 5
    sem := make(chan struct{}, concurrency)

    for _, user := range users {
        wg.Add(1)
        sem <- struct{}{}
        go func(u model.UserProfile) {
            defer wg.Done()
            defer func() { <-sem }()
            s.processUser(ctx, u)
        }(user)
    }

    wg.Wait()
    duration := time.Since(start).Seconds()
    s.SyncDuration.Observe(duration)
}

func (s *SyncService) processUser(ctx context.Context, user model.UserProfile) {
    if !s.EmailRe.MatchString(user.Email) {
        user.SyncStatus = "failed_validation"
        user.SyncMessage = "Invalid email format"
        s.Logger.Warn("Validation failed", zap.String("id", user.ID), zap.String("reason", user.SyncMessage))
        s.Repo.InsertOrUpdateUser(ctx, user)
        s.ProfilesSkipped.Inc()
        return
    }

    if user.Status != "active" && user.Status != "pending" {
        user.SyncStatus = "skipped"
        user.SyncMessage = "Status is neither active nor pending"
        s.Logger.Warn("Skipped user", zap.String("id", user.ID), zap.String("reason", user.SyncMessage))
        s.Repo.InsertOrUpdateUser(ctx, user)
        s.ProfilesSkipped.Inc()
        return
    }

    existing, _ := s.Repo.GetUserByID(ctx, user.ID)
    if existing != nil && existing.LastUpdatedAt >= user.LastUpdatedAt {
        user.SyncStatus = "skipped"
        user.SyncMessage = "Not newer than existing record"
        s.Logger.Info("Skipped older record", zap.String("id", user.ID))
        s.Repo.InsertOrUpdateUser(ctx, user)
        s.ProfilesSkipped.Inc()
        return
    }

    user.SyncStatus = "synced"
    user.SyncMessage = "Successfully synced"
    if err := s.Repo.InsertOrUpdateUser(ctx, user); err != nil {
        s.Logger.Error("DB insert/update failed", zap.Error(err), zap.String("id", user.ID))
    } else {
        s.Logger.Info("User synced", zap.String("id", user.ID))
        s.ProfilesSynced.Inc()
    }
}
