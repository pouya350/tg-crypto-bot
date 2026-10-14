package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TgToken string
	DbURL   string
}

func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, fmt.Errorf("ошибка загрузки .env файла: %w", err)
	}
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DB_URL не найден в окружении")
	}
	token := os.Getenv("TG_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TG_TOKEN не найден в окружении")
	}
	return &Config{TgToken: token,
		 			DbURL: dbURL, 
		 			}, nil
}
