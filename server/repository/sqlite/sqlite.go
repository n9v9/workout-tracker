package sqlite

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

type DB struct {
	*sqlx.DB
}

// NewDB creates a new SQLite connection to the given file and pings it
// to check whether the connection is established successfully.
func NewDB(file string) (*DB, error) {
	args := []string{
		"_pragma=foreign_keys(1)", // Enable foreign key checking.
	}

	db, err := sqlx.Open("sqlite", file+"?"+strings.Join(args, "&"))
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to test connection to database: %w", err)
	}

	return &DB{db}, nil
}

// RunMigrations runs all remaining `up` migrations.
func (db *DB) RunMigrations(migrations fs.FS) error {
	log.Info().Msg("Running migrations.")
	start := time.Now()
	defer func() {
		log.Info().Dur("duration", time.Since(start)).Msg("Running migrations done.")
	}()

	driver, err := sqlite.WithInstance(db.DB.DB, new(sqlite.Config))
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	files, err := iofs.New(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create iofs source driver for migrations: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", files, "workout-tracker", driver)
	if err != nil {
		return fmt.Errorf("failed to create migration instance: %w", err)
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info().Msg("All migrations are already applied.")
		} else {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	return nil
}
