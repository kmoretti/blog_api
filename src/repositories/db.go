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

	migrationFiles, err := collectMigrationFiles()
	if err != nil {
		return nil, fmt.Errorf("could not find migration files: %w", err)
	}

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

		// Split by semicolons respecting BEGIN...END blocks (triggers, procedures)
		statements := splitSQLStatements(string(content))
		if err := db.Transaction(func(tx *gorm.DB) error {
			for _, stmt := range statements {
				stmt = strings.TrimSpace(stmt)
				if stmt == "" {
					continue
				}
				if err := tx.Exec(stmt).Error; err != nil {
					// Ignore "duplicate column" errors for ALTER TABLE (migration re-run safety)
					if strings.Contains(err.Error(), "duplicate column name") {
						log.Printf("[Repo]跳过已存在的列: %v", err)
						continue
					}
					return err
				}
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

// collectMigrationFiles returns the ordered list of SQL migration files to apply.
// Primary migrations come from migrations/*.sql (sorted alphabetically), and
// manual migrations come from migrations/manual/*.sql (sorted alphabetically),
// running after the primary migrations so they can alter columns created earlier.
// Deduplication by basename across both directories is enforced; only one of
// each basename is kept (the primary one wins).
func collectMigrationFiles() ([]string, error) {
	primary, err := filepath.Glob("migrations/*.sql")
	if err != nil {
		return nil, err
	}
	manual, err := filepath.Glob("migrations/manual/*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(primary)
	sort.Strings(manual)

	seen := make(map[string]bool, len(primary)+len(manual))
	files := make([]string, 0, len(primary)+len(manual))
	for _, f := range primary {
		name := filepath.Base(f)
		if seen[name] {
			continue
		}
		seen[name] = true
		files = append(files, f)
	}
	for _, f := range manual {
		name := filepath.Base(f)
		if seen[name] {
			continue
		}
		seen[name] = true
		files = append(files, f)
	}
	return files, nil
}

// splitSQLStatements splits SQL content by semicolons, respecting BEGIN...END blocks
// so that trigger/procedure bodies with internal semicolons are kept intact.
func splitSQLStatements(sql string) []string {
	var result []string
	depth := 0
	start := 0
	runes := []rune(sql)
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(runes); i++ {
		ch := runes[i]

		// Track line comments
		if !inBlockComment && ch == '-' && i+1 < len(runes) && runes[i+1] == '-' {
			inLineComment = true
			continue
		}
		if inLineComment && ch == '\n' {
			inLineComment = false
			continue
		}
		if inLineComment {
			continue
		}

		// Track block comments
		if !inLineComment && ch == '/' && i+1 < len(runes) && runes[i+1] == '*' {
			inBlockComment = true
			i++ // skip *
			continue
		}
		if inBlockComment && ch == '*' && i+1 < len(runes) && runes[i+1] == '/' {
			inBlockComment = false
			i++ // skip /
			continue
		}
		if inBlockComment {
			continue
		}

		// Track BEGIN/END depth (case-insensitive)
		if ch == 'B' || ch == 'b' {
			if i+4 < len(runes) && toUpperStr(string(runes[i:i+5])) == "BEGIN" {
				// Make sure it's a word boundary
				if i+5 >= len(runes) || !isIdentChar(runes[i+5]) {
					depth++
					i += 4
					continue
				}
			}
		}
		if ch == 'E' || ch == 'e' {
			if i+2 < len(runes) && toUpperStr(string(runes[i:i+3])) == "END" {
				if i+3 >= len(runes) || !isIdentChar(runes[i+3]) {
					depth--
					i += 2
					continue
				}
			}
		}

		// Split on semicolons only when not inside BEGIN...END
		if ch == ';' && depth == 0 {
			stmt := strings.TrimSpace(string(runes[start : i+1]))
			if stmt != "" {
				result = append(result, stmt)
			}
			start = i + 1
		}
	}

	// Remaining
	if start < len(runes) {
		stmt := strings.TrimSpace(string(runes[start:]))
		if stmt != "" {
			result = append(result, stmt)
		}
	}

	return result
}

func toUpperStr(s string) string {
	return strings.ToUpper(s)
}

func isIdentChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
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
