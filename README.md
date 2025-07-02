*** Teqbae Data Synchronization Microservices ***

This repository contains two Go microservices built for Teqbae as a machine task.
It simulates data synchronization between a SourceService and a TargetService, with robust concurrency, validation, PostgreSQL persistence, and Prometheus metrics.

## Project Overview

### Services

#### SourceService
- Simulates a microservice generating user profile data.
- Exposes an HTTP endpoint that returns a batch of new or updated user profiles on each call.

#### TargetService
- Periodically fetches data from SourceService.
- Applies business validation:
  - Valid email format.
  - Only active or pending statuses.
  - Ensures newer updates based on `last_updated_at`.
- Persists valid data into PostgreSQL with proper `sync_status` and `sync_message`.
- Exposes Prometheus metrics for monitoring.

## Getting Started

### Prerequisites
- Docker + Docker Compose installed.

  This project uses Docker to run both services and PostgreSQL.
  Make sure you have Docker installed on your system.
  On Windows & Mac, install Docker Desktop.
  On Linux, install docker and docker-compose packages using your distro’s package manager.
  (Optional) Git
  If you want to clone these repositories.

  Note: This project is tested on Docker Desktop on Windows, but it will also work on Linux or MacOS with Docker installed.

## Running the application

### 1. Clone the repository
```bash
git clone <https://github.com/Prototype-1/SourceService> <https://github.com/Prototype-1/TargetService>
cd Prototype-1
```

### 2. Start all services with Docker Compose
```bash
docker-compose up --build
                OR
docker-compose up (if already built)            
```

This will:
- Start a PostgreSQL database.
- Build and start SourceService on port 8080.
- Build and start TargetService on port 2112 (for Prometheus metrics).

## PostgreSQL Database

### Connect to PostgreSQL running in Docker
```bash
docker exec -it targetservice-postgres-1 psql -U postgres -d target_service
```

### Database Schema
The TargetService automatically runs migrations on startup to ensure the users table exists.

#### SQL schema:
```sql
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    mobile VARCHAR(20) UNIQUE NOT NULL,
    status VARCHAR(50) NOT NULL,
    last_updated_at TIMESTAMPTZ NOT NULL,
    sync_status VARCHAR(50) NOT NULL DEFAULT 'synced',
    sync_message TEXT
);
```

## API Endpoints

### SourceService

#### Get user changes
```
GET /users/changes
```

- Returns a JSON array of user profiles.
- Example:
```json
[
  {
    "id": "uuid-string",
    "name": "User123",
    "email": "user123@example.com",
    "mobile": "+911234567890",
    "status": "active",
    "last_updated_at": "2025-07-01T15:43:04+05:30"
  }
]
```
- Accessible on: http://localhost:8080/users/changes

## Monitoring

### TargetService Prometheus Metrics
- Exposed on http://localhost:2112/metrics.

**Key metrics:**
- `profiles_fetched_total`: total profiles fetched from SourceService.
- `profiles_synced_total`: total profiles successfully synced to DB.
- `profiles_skipped_total`: profiles skipped due to validation or duplicate checks.
- `sync_batch_duration_seconds`: histogram of time taken per batch.

## Configuration

The TargetService uses a `.env` file (loaded by `github.com/joho/godotenv`) for configuration.

**Example .env:**
```env
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=your_db_name
SOURCE_SERVICE_URL=http://source-service:8080/users/changes
SYNC_INTERVAL=15
```

## Teqbae Coding Standards & Highlights

- **Concurrency** with goroutines, wait groups, and a semaphore to limit DB writes.
- Uses **structured logging** via Uber's zap logger.
- **Prometheus client** integrated for metrics.
- **Idempotent updates** with `last_updated_at` checks.
- **Auto-migration** ensures database schema is up to date.

## Running Manually

### SourceService
```bash
cd SourceService
go run main.go
```

### TargetService
```bash
cd TargetService
go run main.go
```

## Notes

- The entire solution is **containerized** and can be deployed in any environment supporting Docker and PostgreSQL.
- Database volume is **persisted** via Docker volumes.

---

*Developed for Teqbae*