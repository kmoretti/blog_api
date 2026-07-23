package repositories

import (
	"blog_api/src/model"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

	db, err := gorm.Open(sqlite.Open(sqliteDSN(dbPath)), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
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

	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name TEXT PRIMARY KEY,
			applied_at INTEGER NOT NULL
		)
	`).Error; err != nil {
		return nil, fmt.Errorf("could not initialize migration history: %w", err)
	}

	for _, file := range migrationFiles {
		name := filepath.Base(file)
		var applied int64
		if err := db.Model(&schemaMigration{}).Where("name = ?", name).Count(&applied).Error; err != nil {
			return nil, fmt.Errorf("could not check migration %s: %w", file, err)
		}
		if applied > 0 {
			continue
		}

		log.Printf("[Repo]运行迁移: %s\n", file)
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("could not read migration file %s: %w", file, err)
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(content)).Error; err != nil {
				return err
			}
			return tx.Create(&schemaMigration{
				Name:      name,
				AppliedAt: time.Now().Unix(),
			}).Error
		}); err != nil {
			return nil, fmt.Errorf("could not execute migration statement in file %s: %w", file, err)
		}
	}

	log.Println("Database migrations completed successfully.")
	return db, nil
}

func sqliteDSN(path string) string {
	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}
	return path + separator + "_foreign_keys=on&_busy_timeout=5000&_journal_mode=WAL&_synchronous=FULL&_secure_delete=on"
}

type schemaMigration struct {
	Name      string `gorm:"column:name;primaryKey"`
	AppliedAt int64  `gorm:"column:applied_at"`
}

func (schemaMigration) TableName() string {
	return "schema_migrations"
}
