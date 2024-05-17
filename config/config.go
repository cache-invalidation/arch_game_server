package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Port   string `env:"PORT" envDefault:"8080"`
	DbHost string `env:"DB_HOST" envDefault:""`
	DbUser string `env:"DB_USER" envDefault:""`
	DbPass string `env:"DB_PASS" envDefault:""`
}

func ReadConfig() (*Config, error) {
	config := Config{}

	if err := env.Parse(&config); err != nil {
		return nil, fmt.Errorf("read config error: %w", err)
	}

	return &config, nil
}
