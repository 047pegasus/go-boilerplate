package tests

import (
	"time"

	"github.com/047pegasus/go-boilerplate/internal/config"
	"github.com/047pegasus/go-boilerplate/internal/database"
	"github.com/047pegasus/go-boilerplate/internal/server"
	"github.com/rs/zerolog"
)

// CreateTestServer creates a server instance for testing
func CreateTestServer(logger *zerolog.Logger, db *TestDB) *server.Server {
	// Set up observability config with defaults if not present
	if db.Config.Observability == nil {
		db.Config.Observability = &config.ObservabilityConfig{
			ServiceName: "alfred-test",
			Environment: "test",
			Logging: config.LoggingConfig{
				Level:                  "info",
				Format:                 "json",
				SlowQueryThresholdTime: 100 * time.Millisecond,
			},
			Sentry: config.SentryConfig{
				Dsn:                 "",    // Empty for tests
				SendDefaultPII:      false, // Disabled for tests
				EnableTracing:       false, // Disabled for tests
				TracesSampleRate:    0,     // Disabled for tests
				DebugLoggingEnabled: true,
			},
			HealthChecks: config.HealthChecksConfig{
				Enabled: false,
			},
		}
	}

	testServer := &server.Server{
		Logger: logger,
		DB: &database.Database{
			Pool: db.Pool,
		},
		Config: db.Config,
	}

	return testServer
}
