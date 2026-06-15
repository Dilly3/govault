package config

import (
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
)

type Environment string

const (
	SandboxEnv Environment = "sandbox"
	ProdEnv  Environment = "prod"
)

type Config struct {
	Env        Environment `envconfig:"ENV"`
	JWTKey     string      `envconfig:"JWT_KEY"`
	Port       string      `envconfig:"PORT"`
	DBUsername string      `envconfig:"DB_USERNAME"`
	DBPassword string      `envconfig:"DB_PASSWORD"`
	DBName     string      `envconfig:"DB_NAME"`
	DBHost     string      `envconfig:"DB_HOST"`
	DBPort     string      `envconfig:"DB_PORT"`
}

var config *Config

func LoadConfig() error {
	var c Config
	err := envconfig.Process("govault", &c)
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}
	envPort := os.Getenv("PORT")
	if envPort != "" {
		config.Port = envPort
	}
	config = &c
	return nil
}

func GetConfig() *Config {
	return config
}
