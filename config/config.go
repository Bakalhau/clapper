package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken        string
	TMDBAPIKey          string
	SuggestionChannelID string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	channelID := os.Getenv("SUGGESTION_CHANNEL_ID")
	if _, err := strconv.ParseInt(channelID, 10, 64); err != nil {
		log.Fatal("Invalid SUGGESTION_CHANNEL_ID: must be a valid integer")
	}

	return &Config{
		DiscordToken:        os.Getenv("DISCORD_TOKEN"),
		TMDBAPIKey:          os.Getenv("TMDB_API_KEY"),
		SuggestionChannelID: channelID,
	}
}