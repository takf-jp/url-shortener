package testutil

import (
	"context"
	"os"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("TEST_DATABASE_URL environment variable is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to get sql.DB from gorm.DB: %v", err)
	}
	if err := sqlDB.PingContext(context.Background()); err != nil {
		t.Fatalf("failed to ping test database: %v", err)
	}

	t.Cleanup(func() {
		sqlDB.Close()
	})

	return db
}

func TruncateAll(t *testing.T, db *gorm.DB) {
	t.Helper()

	if err := db.Exec("TRUNCATE TABLE urls, users RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
}
