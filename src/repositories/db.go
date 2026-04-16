package repositories

import (
	"blog_api/src/model"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// InitDB initializes the database and runs migrations.
func InitDB(cfg *model.Config) (*gorm.DB, error) {
	dbPath := cfg.Data.Database.Path
	if dbPath == "" {
		return nil, fmt.Errorf("database path is not configured")
	}

	log.Printf("初始化数据库于: %s", dbPath)

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error
		},
	)

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	// Enable foreign keys
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, fmt.Errorf("could not enable foreign keys: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("could not get sql.DB from gorm: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("could not connect to database via gorm: %w", err)
	}

	migrationFiles, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("could not find migration files: %w", err)
	}
	sort.Strings(migrationFiles)

	for _, file := range migrationFiles {
		log.Printf("[Repo]运行迁移: %s\n", file)
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("could not read migration file %s: %w", file, err)
		}

		// Execute the entire migration file content at once
		if err := db.Exec(string(content)).Error; err != nil {
			return nil, fmt.Errorf("could not execute migration statement in file %s: %w", file, err)
		}
	}

	log.Println("Database migrations completed successfully.")
	return db, nil
}
