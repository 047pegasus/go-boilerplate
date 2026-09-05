package config

import (
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	_ "github.com/joho/godotenv/autoload"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"
	"github.com/rs/zerolog"
)

type Config struct {
	Primary       Primary              `koanf:"primary" validate:"required"`
	Server        ServerConfig         `koanf:"server" validate:"required"`
	Database      DatabaseConfig       `koanf:"database" validate:"required"`
	Cache         CacheConfig          `koanf:"cache" validate:"required"`
	Auth          AuthConfig           `koanf:"auth" validate:"required"`
	Integrations  IntegrationConfig    `json:"integrations" validate:"required"`
	Observability *ObservabilityConfig `koanf:"observability"`
}

type Primary struct {
	Env string `koanf:"env" validate:"required"`
}

type ServerConfig struct {
	Port               string   `koanf:"server_port" validate:"required"`
	ReadTimeout        int      `koanf:"read_timeout" validate:"required"`
	WriteTimeout       int      `koanf:"write_timeout" validate:"required"`
	IdleTimeout        int      `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string `koanf:"cors_allowed_origins" validate:"required"`
}

type DatabaseConfig struct {
	Host            string `koanf:"db_host" validate:"required"`
	Port            int    `koanf:"db_port" validate:"required"`
	User            string `koanf:"db_user" validate:"required"`
	Password        string `koanf:"db_password" validate:"required"`
	Name            string `koanf:"db_name" validate:"required"`
	SSLMode         string `koanf:"db_ssl_mode" validate:"required"`
	MaxOpenConns    int    `koanf:"db_max_open_conns" validate:"required"`
	MaxIdleConns    int    `koanf:"db_max_idle_conns" validate:"required"`
	ConnMaxLifetime int    `koanf:"db_conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime int    `koanf:"db_conn_max_idle_time" validate:"required"`
}

type IntegrationConfig struct {
	ResendAPIKey string `koanf:"resend_api_key" validate:"required"`
}

type CacheConfig struct {
	Address string `koanf:"cache_address_url"  validate:"required"`
}

// Since we are going to use Clerk as our Authentication tool provider we are making our AuthConfig struct accordingly!!
type AuthConfig struct {
	SecretKey string `koanf:"auth_secret_key" validate:"required"`
}

func LoadConfig() (*Config, error) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	k := koanf.New(".")

	err := k.Load(env.Provider("APP_NAME_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "APP_NAME_"))
	}), nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("Error loading config !!")
	}

	mainConfig := &Config{}
	err = k.Unmarshal("", mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("Error unmarshalling config !!")
	}

	v := validator.New()
	err = v.Struct(mainConfig)
	if err != nil {
		logger.Fatal().Err(err).Msg("Error: validating config failed !!")
	}

	if mainConfig.Observability == nil {
		mainConfig.Observability = DefaultObservabilityConfig()
	}
	mainConfig.Observability.ServiceName = "go-boilerplate"
	mainConfig.Observability.Environment = mainConfig.Primary.Env

	if err := mainConfig.Observability.Validate(); err != nil {
		logger.Fatal().Err(err).Msg("Error validating observability config !!")
	}

	return mainConfig, nil
}
