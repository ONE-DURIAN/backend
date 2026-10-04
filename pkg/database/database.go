package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"community-backend/config"
)

type DB struct {
	*sql.DB
}

func ConnectPostgres(cfg config.Config) (*DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=5",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Connection Pool Settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(15 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Retry connecting up to 5 times (total ~10s) to handle database cold-starts
	var pingErr error
	for attempt := 1; attempt <= 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		pingErr = db.PingContext(ctx)
		cancel()

		if pingErr == nil {
			return &DB{db}, nil
		}

		if attempt < 5 {
			time.Sleep(2 * time.Second)
		}
	}

	_ = db.Close()
	return nil, fmt.Errorf("failed to connect to postgres after 5 attempts: %w", pingErr)
}

func (db *DB) HealthCheck(ctx context.Context) (float64, error) {
	start := time.Now()
	if err := db.PingContext(ctx); err != nil {
		return 0, err
	}
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	return latency, nil
}
