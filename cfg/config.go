package cfg

import (
	"os"
)

type Config struct {
	Port   string
	DBPath string
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not set in environment
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "url_shortener.db" // Default database path if not set in environment
	}

	return &Config{
		Port:   port,
		DBPath: dbPath,
	}
}
