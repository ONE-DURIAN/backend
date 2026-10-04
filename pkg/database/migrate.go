package database

import (
	"context"
	"fmt"
	"log"
	"time"
)

// AutoMigrate runs initial schema migrations on startup
func (db *DB) AutoMigrate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queries := []string{
		// 1. Enable UUID Extension if not already present
		`CREATE EXTENSION IF NOT EXISTS "pgcrypto";`,

		// 2. Users Table
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			phone_number VARCHAR(20),
			email VARCHAR(255),
			password_hash VARCHAR(255) NOT NULL,
			full_name VARCHAR(100) NOT NULL,
			avatar_url TEXT,
			role VARCHAR(20) DEFAULT 'farmer' NOT NULL,
			national_id VARCHAR(13),
			is_active BOOLEAN DEFAULT TRUE NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			deleted_at TIMESTAMPTZ
		);`,

		// 3. Drop legacy inline constraints if upgrading from earlier version
		`ALTER TABLE users DROP CONSTRAINT IF EXISTS users_phone_number_key;`,
		`ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;`,

		// 4. Partial Unique Indexes (Enforces uniqueness ONLY for active accounts, allowing re-registration after soft delete)
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_phone_unique ON users(phone_number) WHERE deleted_at IS NULL;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique ON users(email) WHERE deleted_at IS NULL;`,
	}

	for i, q := range queries {
		if _, err := db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migration step %d failed: %w", i+1, err)
		}
	}

	log.Println("✅ Database Auto-Migration completed successfully (users table ready)")
	return nil
}
