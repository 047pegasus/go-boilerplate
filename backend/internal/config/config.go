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
	Kafka         *KafkaConfig         `koanf:"kafka"`
	ObjectStorage *ObjectStorageConfig `koanf:"object_storage"`
	Auth          AuthConfig           `koanf:"auth" validate:"required"`
	Integrations  IntegrationConfig    `koanf:"integrations" validate:"required"`
	Observability *ObservabilityConfig `koanf:"observability"`
}

type Primary struct {
	Env string `koanf:"env" validate:"required"`
}

type ServerConfig struct {
	Port               string   `koanf:"port" validate:"required"`
	ReadTimeout        int      `koanf:"read_timeout" validate:"required"`
	WriteTimeout       int      `koanf:"write_timeout" validate:"required"`
	IdleTimeout        int      `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string `koanf:"cors_allowed_origins" validate:"required"`
}

type DatabaseConfig struct {
	Host            string `koanf:"host" validate:"required"`
	Port            int    `koanf:"port" validate:"required"`
	User            string `koanf:"user" validate:"required"`
	Password        string `koanf:"password" validate:"required"`
	Name            string `koanf:"name" validate:"required"`
	SSLMode         string `koanf:"ssl_mode" validate:"required"`
	MaxOpenConns    int    `koanf:"max_open_conns" validate:"required"`
	MaxIdleConns    int    `koanf:"max_idle_conns" validate:"required"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime" validate:"required"`
	ConnMaxIdleTime int    `koanf:"conn_max_idle_time" validate:"required"`
}

type IntegrationConfig struct {
	ResendAPIKey string `koanf:"resend_api_key" validate:"required"`
}

// CacheConfig holds the Valkey connection details. Named "cache" rather than
// "redis" since the client was migrated from go-redis to valkey-go.
type CacheConfig struct {
	Address string `koanf:"address" validate:"required"`
}

// KafkaConfig is optional — leave the whole "kafka" section out of your env
// to run without Kafka. If present, brokers/consumer group are required.
type KafkaConfig struct {
	Brokers       []string `koanf:"brokers" validate:"required"`
	ClientID      string   `koanf:"client_id"`
	ConsumerGroup string   `koanf:"consumer_group" validate:"required"`
}

// ObjectStorageConfig targets any S3-compatible provider — MinIO, Cloudflare
// R2, Backblaze B2, or AWS S3. Optional — omit the section to disable.
type ObjectStorageConfig struct {
	Endpoint  string `koanf:"endpoint" validate:"required"`
	Region    string `koanf:"region"`
	AccessKey string `koanf:"access_key" validate:"required"`
	SecretKey string `koanf:"secret_key" validate:"required"`
	Bucket    string `koanf:"bucket" validate:"required"`
	UseSSL    bool   `koanf:"use_ssl"`
}

// Since we are going to use Clerk as our Authentication tool provider we are making our AuthConfig struct accordingly!!
type AuthConfig struct {
	SecretKey string `koanf:"secret_key" validate:"required"`
}

func LoadConfig() (*Config, error) {
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()

	k := koanf.New(".")

	const envPrefix = "BOILERPLATE_"
	err := k.Load(env.Provider(envPrefix, ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, envPrefix))
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
