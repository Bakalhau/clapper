package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken string
	TMDBAPIKey   string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		DiscordToken: os.Getenv("DISCORD_TOKEN"),
		TMDBAPIKey:   os.Getenv("TMDB_API_KEY"),
	}
}