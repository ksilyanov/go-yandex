package config

import (
	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerURL       string `env:"SERVER_ADDRESS" envDefault:"localhost:8080"`
	BaseURL         string `env:"BASE_URL" envDefault:"http://localhost:8080"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" envDefault:"./test.txt"`
	DBDSN           string `env:"DATABASE_DSN" envDefault:""`
}

func GetConfig() (Config, error) {
	config := &Config{}
	err := env.Parse(config)
	if err != nil {
		return *config, err
	}
	return *config, nil
}
