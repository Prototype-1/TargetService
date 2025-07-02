package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"
    "github.com/Prototype-1/TargetService/config"
    "github.com/Prototype-1/TargetService/internal/repository"
    "github.com/Prototype-1/TargetService/internal/service"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    cfg := config.LoadConfig()

    dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
        cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName)
    repo, err := repository.NewUserRepository(dbURL, logger)
    if err != nil {
        logger.Fatal("Could not connect to DB", zap.Error(err))
    }
    defer repo.Close()

    syncSvc := service.NewSyncService(repo, logger, cfg)

    ctx, cancel := context.WithCancel(context.Background())
    go syncSvc.Start(ctx)

    // graceful shutdown using os package
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    <-c
    logger.Info("Shutdown signal received")
    cancel()
    time.Sleep(2 * time.Second)
}
