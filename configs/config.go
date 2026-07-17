// loads env variables into a struct
package configs

import "os"

type Config struct {
	DatabaseURL string
	Port        string
	BankBaseURL string
}

func Load() *Config {
	return &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Port:        os.Getenv("PORT"),
		BankBaseURL: os.Getenv("BANK_BASE_URL"),
	}
}
