package configs

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	RabbitMQ RabbitMQConfig `mapstructure:"rabbitmq"`
	Keycloak KeycloakConfig `mapstructure:"keycloak"`
	CORS     CORSConfig     `mapstructure:"cors"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

type ServerConfig struct {
	Port        int    `mapstructure:"port"`
	Environment string `mapstructure:"environment"`
}

type DatabaseConfig struct {
	URL string `mapstructure:"url"`
}

type RabbitMQConfig struct {
	URL      string            `mapstructure:"url"`
	Exchange string            `mapstructure:"exchange"`
	Queue    RabbitMQQueueConfig `mapstructure:"queue"`
}

type RabbitMQQueueConfig struct {
	Purchased string `mapstructure:"purchased"`
	Confirmed string `mapstructure:"confirmed"`
}

type KeycloakConfig struct {
	URL      string `mapstructure:"url"`
	Realm    string `mapstructure:"realm"`
	ClientID string `mapstructure:"client_id"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	AllowedMethods []string `mapstructure:"allowed_methods"`
	AllowedHeaders []string `mapstructure:"allowed_headers"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Warn("No .env file found, using environment variables")
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")
	viper.AddConfigPath("../../configs")
	viper.AddConfigPath("../../../configs")

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set database URL from environment
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		config.Database.URL = dbURL
	}

	// Set RabbitMQ URL from environment
	if rabbitURL := os.Getenv("RABBITMQ_URL"); rabbitURL != "" {
		config.RabbitMQ.URL = rabbitURL
	}

	// Set Keycloak URL from environment
	if keycloakURL := os.Getenv("KEYCLOAK_URL"); keycloakURL != "" {
		config.Keycloak.URL = keycloakURL
	}

	return &config, nil
}
