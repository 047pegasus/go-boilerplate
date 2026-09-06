package config

import (
	"fmt"
	"slices"
	"time"
)

/*
Using Sentry by default in our project for Observability, Logging and Monitoring Purposes"
NOTE: The Sentry provider can be exchanged & used instead by NewRelic as observability provider by uncommenting the NewRelicConfig Struct implementations
*/
type ObservabilityConfig struct {
	ServiceName string        `koanf:"service_name" validate:"required"`
	Environment string        `koanf:"environment" validate:"required"`
	Logging     LoggingConfig `koanf:"logging" validate:"required"`
	Sentry      SentryConfig  `koanf:"sentry" validate:"required"`
	//NewRelic     NewRelicConfig     `koanf:"new_relic" validate:"required"`
	HealthChecks HealthChecksConfig `koanf:"health_checks" validate:"required"`
}

type LoggingConfig struct {
	Level                  string        `koanf:"level" validate:"required"`
	Format                 string        `koanf:"format" validate:"required"`
	SlowQueryThresholdTime time.Duration `koanf:"slow_query_threshold"`
}

// ------------------------SentryConfig----------------------------------------
type SentryConfig struct {
	// Dsn is intentionally NOT required: an empty DSN is how the app disables
	// Sentry entirely (see LoggerService.NewLoggerService's `if cfg.Sentry.Dsn == ""` check).
	Dsn                 string  `koanf:"dsn"`
	SendDefaultPII      bool    `koanf:"send_default_pii"`
	EnableTracing       bool    `koanf:"enable_tracing"`
	TracesSampleRate    float64 `koanf:"trace_sample_rate"`
	DebugLoggingEnabled bool    `koanf:"debug_logging_enabled"`
	//EnableLogs          bool    `koanf:"enable_logs"`
	//EnableMetrics       bool    `koanf:"enable_metrics"`
}

/*
Note: Uncomment this to use NewRelic instead !!
// ----------------NewRelicConfig---------------------------------------------
type NewRelicConfig struct {
	LicenseKey                string `koanf:"license_key" validate:"required"`
	AppLogForwardingEnabled   bool   `koanf:"app_log_forwarding_enabled"`
	DistributedTracingEnabled bool   `koanf:"distributed_tracing_enabled"`
	DebugLoggingEnabled       bool   `koanf:"debug_logging_enabled"`
}
*/

type HealthChecksConfig struct {
	Enabled  bool          `koanf:"enabled" validate:"required"`
	Interval time.Duration `koanf:"interval" validate:"min=1s"`
	Timeout  time.Duration `koanf:"timeout" validate:"min=1s"`
	Checks   []string      `koanf:"checks"`
}

func DefaultObservabilityConfig() *ObservabilityConfig {
	return &ObservabilityConfig{
		ServiceName: "go-boilerplate",
		Environment: "development",
		Logging: LoggingConfig{
			Level:                  "info",
			Format:                 "json",
			SlowQueryThresholdTime: 100 * time.Millisecond,
		},
		Sentry: SentryConfig{
			Dsn:                 "",
			SendDefaultPII:      true,
			EnableTracing:       true,
			TracesSampleRate:    0.5, // by default capture 50% transactions for tracing
			DebugLoggingEnabled: false,
			//EnableLogs:          true,
			//EnableMetrics:       true,
		},
		HealthChecks: HealthChecksConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
			Checks:   []string{"database", "cache"}, // add more service checks like queues etc.. in this when introducing them in the application
		},
	}
}

func (cfg *ObservabilityConfig) Validate() error {
	if cfg.ServiceName == "" {
		return fmt.Errorf("service_name is required and(or) cannot be empty !!")
	}
	if cfg.Environment == "" {
		return fmt.Errorf("environment is required and cannot be empty !!")
	}

	// "local" is included alongside the standard deploy environments since
	// Primary.Env == "local" is used elsewhere (e.g. database.go) to gate
	// local-only behavior like verbose pgx query logging.
	validEnvs := []string{"production", "staging", "development", "local"}
	if !slices.Contains(validEnvs, cfg.Environment) {
		return fmt.Errorf("environment '%s' is not valid", cfg.Environment)
	}

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[cfg.Logging.Level] {
		return fmt.Errorf("invalid log_level: %s", cfg.Logging.Level)
	}

	// Check for preventing production from having debug logging enabled
	if cfg.Logging.Level == "debug" && cfg.IsProduction() {
		return fmt.Errorf("invalid log_level for environment: %s %s \n debug logs only allowed for development", cfg.Logging.Level, cfg.Environment)
	}

	if cfg.Logging.SlowQueryThresholdTime <= 0 {
		return fmt.Errorf("slow_query_threshold_time must be greater than zero !!")
	}

	return nil
}

func (cfg *ObservabilityConfig) GetLogLevel() string {
	switch cfg.Environment {
	case "production":
		if cfg.Logging.Level == "" {
			return "info"
		}
	case "development", "local":
		if cfg.Logging.Level == "" {
			return "debug"
		}
	}
	return cfg.Logging.Level
}

func (cfg *ObservabilityConfig) IsProduction() bool {
	return cfg.Environment == "production"
}
