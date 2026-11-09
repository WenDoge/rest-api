package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL  string
	Port         string
	JWTSecret    string
	CookieSecure bool
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {

		log.Print(err)

		return nil, err
	}

	config := &Config{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		Port:         os.Getenv("PORT"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		CookieSecure: os.Getenv("COOKIE_SECURE") == "true",
	}

	return config, nil
}
