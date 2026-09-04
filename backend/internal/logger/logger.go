package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/047pegasus/go-boilerplate/internal/config"
	"github.com/getsentry/sentry-go"
	"github.com/getsentry/sentry-go/zerolog"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

//NOTE: MODIFY ALL CALLS FOR NEW RELIC INTEGRATION in places left for per user integration such as in function signature returns
//WILL UPDATE IN FUTURE AND SEPARATE NEW RELIC AND OTHER OBSERVABILITY TOOLS SEPARATELY

// LoggerService manages Sentry/New Relic integration and logger creation
type LoggerService struct {
	sentryApp *sentry.Client
	//nrApp *newrelic.Application //NewRelic App Instance
}

// NewLoggerService creates a new logger service with Sentry/New Relic integration
func NewLoggerService(cfg *config.ObservabilityConfig) *LoggerService {
	service := &LoggerService{}
	if cfg.Sentry.Dsn == "" {
		return service // simply return service if we need not initialize logging, we know this by the fact that the sentry dsn is not provided
	}
	/*
		if cfg.NewRelic.LicenseKey== "" {
			return service // simply return service if we need not initialize logging, we know this by the fact that the sentry dsn is not provided
		}
	*/

	var ConfigOptions sentry.ClientOptions
	ConfigOptions.Dsn = cfg.Sentry.Dsn
	ConfigOptions.SendDefaultPII = cfg.Sentry.SendDefaultPII
	ConfigOptions.EnableTracing = cfg.Sentry.EnableTracing
	// Set TracesSampleRate to 1.0 to capture 100% of transactions for tracing.
	ConfigOptions.TracesSampleRate = cfg.Sentry.TracesSampleRate

	if cfg.Sentry.DebugLoggingEnabled {
		ConfigOptions.Debug = true
	}

	app, err := sentry.NewClient(ConfigOptions)
	defer sentry.Flush(2 * time.Second)
	if err != nil {
		log.Error().Err(err).Msg("sentry initialization failed")
		return service
	}
	service.sentryApp = app

	/* ------------------------ NEW RELIC Integration parts ---------------------------
	var configOptions []newrelic.ConfigOption
	configOptions = append(configOptions,
		newrelic.ConfigAppName(cfg.ServiceName),
		newrelic.ConfigLicense(cfg.NewRelic.LicenseKey),
		newrelic.ConfigAppLogForwardingEnabled(cfg.NewRelic.AppLogForwardingEnabled),
		newrelic.ConfigDistributedTracerEnabled(cfg.NewRelic.DistributedTracingEnabled),
	)

	// Add debug logging only if explicitly enabled
	if cfg.NewRelic.DebugLogging {
		configOptions = append(configOptions, newrelic.ConfigDebugLogger(os.Stdout))
	}

	app, err := newrelic.NewApplication(configOptions...)
	if err != nil {
		return service
	}

	service.nrApp = app
	*/
	return service
}

func (ls *LoggerService) Shutdown() {
	if ls.sentryApp != nil {
		ls.sentryApp.Flush(10 * time.Second)
	}
}

func (ls *LoggerService) GetApplication() *sentry.Client {
	return ls.sentryApp
}

// NewLoggerWithService creates a logger with full config and logger service
func NewLoggerWithService(cfg *config.ObservabilityConfig, ls *LoggerService) zerolog.Logger {
	var logLevel zerolog.Level
	level := cfg.GetLogLevel()

	switch level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	default:
		logLevel = zerolog.InfoLevel
	}
	// Don't set global level - let each logger have its own level
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	var writer io.Writer

	// Setup base writer
	var baseWriter io.Writer
	if cfg.IsProduction() && cfg.Logging.Format == "json" {
		// In production, write to stdout
		baseWriter = os.Stdout
		// Wrap with Sentry/New Relic zerologWriter for log forwarding in production
		if ls != nil && ls.sentryApp != nil {
			sentryWriter, err := sentryzerolog.New(sentryzerolog.Config{
				Options: sentryzerolog.Options{
					// These levels will be sent as error events
					Levels: []zerolog.Level{zerolog.ErrorLevel, zerolog.FatalLevel, zerolog.PanicLevel},
					// Disable breadcrumbs in concurrent applications to prevent leakage
					WithBreadcrumbs: false,
					FlushTimeout:    3 * time.Second,
				},
			})
			if err != nil {
				log.Fatal().Err(err).Msg("failed to create sentry writer")
			}
			defer sentryWriter.Close()
			writer = sentryWriter
		} else {
			writer = baseWriter
		}
	} else {
		// Development mode - use console writer
		consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "2006-01-02 15:04:05"}
		writer = consoleWriter
	}

	logger := zerolog.New(writer).
		Level(logLevel).
		With().
		Timestamp().
		Str("service", cfg.ServiceName).
		Str("environment", cfg.Environment).
		Logger()

	// Include stack traces for errs in development
	if !cfg.IsProduction() {
		logger = logger.With().Stack().Logger()
	}

	return logger
}

// WithTraceContext adds New Relic transaction context to logger
func WithTraceContext(logger zerolog.Logger, span *sentry.Span) zerolog.Logger {
	if span == nil {
		return logger
	}
	//Get trace metadata from transaction
	return logger.With().
		Str("Trace ID:", span.TraceID.String()).
		Str("Span ID:", span.SpanID.String()).
		Logger()
}

// NewPgxLogger creates a database logger
func NewPgxLogger(level zerolog.Level) zerolog.Logger {
	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
		FormatFieldValue: func(i any) string {
			switch v := i.(type) {
			case string:
				if len(v) > 200 {
					return v[:200] + "..."
				}
				return v
			case []byte:
				var obj interface{}
				if err := json.Unmarshal(v, &obj); err == nil {
					pretty, _ := json.MarshalIndent(obj, "", "	")
					return "\n" + string(pretty) + "\n"
				}
				return string(v)
			default:
				return fmt.Sprintf("%v", v)
			}
		},
	}

	return zerolog.New(writer).Level(level).With().Timestamp().Str("component", "database").Logger()
}

// GetPgxTraceLogLevel converts zerolog level to pgx tracelog level
func GetPgxTraceLogLevel(level zerolog.Level) int {
	switch level {
	case zerolog.DebugLevel:
		return 6 // tracelog.LogLevelDebug
	case zerolog.InfoLevel:
		return 4 // tracelog.LogLevelInfo
	case zerolog.WarnLevel:
		return 3 // tracelog.LogLevelWarn
	case zerolog.ErrorLevel:
		return 2 // tracelog.LogLevelError
	default:
		return 0 // tracelog.LogLevelNone
	}
}
