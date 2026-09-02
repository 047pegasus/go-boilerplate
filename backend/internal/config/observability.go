package config

import (
	"fmt"
	"slices"
	"time"
)

/*
Using Sentry by default in our project for Observability, Logging and Monitoring Purposes"
NOTE: The Sentry provider can be exchanged &  used instead by NewRelic as observability provider by uncommenting the NewRelicConfig Struct implementations
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
	Level                  string        `koanf:"log_level" validate:"required"`
	Format                 string        `koanf:"log_format" validate:"required"`
	SlowQueryThresholdTime time.Duration `koanf:"log_slow_query_threshold_time"`
}

// ------------------------SentryConfig----------------------------------------
type SentryConfig struct {
	Dsn                       string `koanf:"obs_sentry_dsn" validate:"required"`
	AppLogForwardingEnabled   bool   `koanf:"obs_app_log_forwarding_enabled"`
	DistributedTracingEnabled bool   `koanf:"obs_distributed_tracing_enabled"`
	DebugLoggingEnabled       bool   `koanf:"obs_debug_logging_enabled"`
}

/*
Note: Uncomment this to use NewRelic instead !!
// ----------------NewRelicConfig---------------------------------------------
type NewRelicConfig struct {
	LicenseKey                string `koanf:"obs_license_key" validate:"required"`
	AppLogForwardingEnabled   bool   `koanf:"obs_app_log_forwarding_enabled"`
	DistributedTracingEnabled bool   `koanf:"obs_distributed_tracing_enabled"`
	DebugLoggingEnabled       bool   `koanf:"obs_debug_logging_enabled"`
}
*/

type HealthChecksConfig struct {
	Enabled  bool          `koanf:"healthcheck_enabled" validate:"required"`
	Interval time.Duration `koanf:"healthcheck_interval" validate:"min=1s"`
	Timeout  time.Duration `koanf:"healthcheck_timeout" validate:"min=1s"`
	Checks   []string      `koanf:"healthcheck_checks"`
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
			Dsn:                       "",
			AppLogForwardingEnabled:   true,
			DistributedTracingEnabled: true,
			DebugLoggingEnabled:       false,
		},
		HealthChecks: HealthChecksConfig{
			Enabled:  true,
			Interval: 30 * time.Second,
			Timeout:  5 * time.Second,
			Checks:   []string{"database", "redis"}, //add more service checks like queues etc.. in this when introducing them in the application
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

	validEnvs := []string{"production", "staging", "development"}
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

	//Check for preventing production from having debug logging enabled
	if cfg.Logging.Level == "debug" && cfg.isProduction() {
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
	case "development":
		if cfg.Logging.Level == "" {
			return "debug"
		}
	}
	return cfg.Logging.Level
}

func (cfg *ObservabilityConfig) isProduction() bool {
	return cfg.Environment == "production"
}
