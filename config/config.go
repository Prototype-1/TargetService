package config

import (
    "log"
    "os"
    "strconv"
    "github.com/joho/godotenv"
)

type Config struct {
    SourceURL   string
    DBUrl       string
    SyncInterval int
}

func Load() Config {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file is available for loading, please make sure the module is there")
    }

    interval, err := strconv.Atoi(os.Getenv("SYNC_INTERVAL"))
    if err != nil {
        interval = 15 
    }

    return Config{
        SourceURL:    os.Getenv("SOURCE_URL"),
        DBUrl:        os.Getenv("DB_URL"),
        SyncInterval: interval,
    }
}
