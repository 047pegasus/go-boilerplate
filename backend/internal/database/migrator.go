package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/url"
	"strconv"

	"github.com/047pegasus/go-boilerplate/internal/config"
	"github.com/jackc/pgx/v5"
	tern "github.com/jackc/tern/v2/migrate"
	"github.com/rs/zerolog"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, logger *zerolog.Logger, cfg *config.Config) error {
	hostport := net.JoinHostPort(cfg.Database.Host, strconv.Itoa(cfg.Database.Port))
	EncodedPassword := url.QueryEscape(cfg.Database.Password)
	connstr := fmt.Sprint("postgres://%s:%s@%s/%s?sslmode=%s",
		cfg.Database.User,
		EncodedPassword,
		hostport,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	conn, err := pgx.Connect(ctx, connstr)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	migrator, err := tern.NewMigrator(ctx, conn, "schema_version")
	if err != nil {
		return fmt.Errorf("Error creating DB migrator: %w", err)
	}
	subtree, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("Error retreiving embedded migrations subtree: %w", err)
	}

	//LOAD MIGRATIONS
	if err := migrator.LoadMigrations(subtree); err != nil {
		return fmt.Errorf("Error loading migrations: %w", err)
	}

	from, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("Error getting current migration version: %w", err)
	}

	if err := migrator.Migrate(ctx); err != nil {
		return err
	}
	if from == int32(len(migrator.Migrations)) {
		logger.Info().Msgf("Database schema up to date, version %d !!", len(migrator.Migrations))
	} else {
		logger.Info().Msgf("Migrated database schema, from %d to %d !!", from, len(migrator.Migrations))
	}

	return nil
}
