package config

import (
    "log"
    "os"
    "strconv"
    "github.com/joho/godotenv"
)

type Config struct {
    DBHost     string
    DBPort     string
    DBUser     string
    DBPassword string
    DBName     string
    SourceURL   string
    SyncInterval int
}

func LoadConfig() Config {
    err := godotenv.Load()
    if err != nil {
        log.Println("No .env file is available for loading, please make sure the module is there")
    }

    syncInterval, err := strconv.Atoi(getEnv("SYNC_INTERVAL", "15"))
    if err != nil {
        syncInterval = 15
    }

    return Config{
        DBHost:       getEnv("DB_HOST", "localhost"),
        DBPort:       getEnv("DB_PORT", "5432"),
        DBUser:       getEnv("DB_USER", "postgres"),
        DBPassword:   getEnv("DB_PASSWORD", ""),
        DBName:       getEnv("DB_NAME", "target_service"),
        SourceURL:    getEnv("SOURCE_SERVICE_URL", "http://localhost:8080/users/changes"),
        SyncInterval: syncInterval,
    }
}

func getEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}
