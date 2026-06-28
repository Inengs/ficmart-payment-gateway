package configs

import "os"

type Config struct {
	DatabaseURL string
	Port string
}

func Load() *Config {
	return &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port: os.Getenv("PORT"),
	}
}