package bot

import (
	"clapper/config"
	"clapper/database"
	"clapper/tmdb"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	session *discordgo.Session
	db      *database.Database
	tmdb    *tmdb.Client
	config  *config.Config
}

func New(cfg *config.Config) (*Bot, error) {
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return nil, fmt.Errorf("error creating Discord session: %w", err)
	}

	db, err := database.New("suggestions.db")
	if err != nil {
		return nil, fmt.Errorf("error initializing database: %w", err)
	}

	return &Bot{
		session: session,
		db:      db,
		tmdb:    tmdb.NewClient(cfg.TMDBAPIKey),
		config:  cfg,
	}, nil
}

func (b *Bot) Start() error {
	b.session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("%s is now online!", s.State.User.Username)
		log.Printf("Bot ID: %s", s.State.User.ID)
	})

	if err := b.session.Open(); err != nil {
		return fmt.Errorf("error opening Discord connection: %w", err)
	}

	if err := b.registerCommands(); err != nil {
		return fmt.Errorf("error registering commands: %w", err)
	}

	return nil
}

func (b *Bot) Stop() {
	b.session.Close()
	b.db.Close()
}

func (b *Bot) registerCommands() error {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "suggestion",
			Description: "Suggest a movie for your server using movie name or TMDB link",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "movie_name",
					Description: "Movie name or TMDB link (e.g., https://www.themoviedb.org/movie/74)",
					Required:    true,
				},
			},
		},
		{
			Name:        "mystats",
			Description: "View your movie suggestion statistics",
		},
		{
			Name:        "mysuggestions",
			Description: "View all your movie suggestions",
		},
		{
			Name:        "pickmovie",
			Description: "Pick a random movie from the suggestions",
		},
		{
			Name:        "moviestats",
			Description: "View overall movie suggestion statistics",
		},
		{
			Name:        "removesuggestion",
			Description: "Remove a movie suggestion",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "movie_name",
					Description: "The name of the movie to remove",
					Required:    true,
				},
			},
		},
	}

	b.session.AddHandler(b.handleInteraction)

	for _, cmd := range commands {
		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd)
		if err != nil {
			return fmt.Errorf("error creating command %s: %w", cmd.Name, err)
		}
		log.Printf("Registered command: %s", cmd.Name)
	}

	return nil
}

func (b *Bot) handleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "suggestion":
			b.handleSuggestion(s, i)
		case "mystats":
			b.handleMyStats(s, i)
		case "mysuggestions":
			b.handleMySuggestions(s, i)
		case "pickmovie":
			b.handlePickMovie(s, i)
		case "moviestats":
			b.handleMovieStats(s, i)
		case "removesuggestion":
			b.handleRemoveSuggestion(s, i)
		}
	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID
		if strings.HasPrefix(customID, "reroll_movie_") {
			b.handleRerollMovie(s, i)
		} else if strings.HasPrefix(customID, "confirm_movie_") {
			b.handleConfirmMovie(s, i)
		} else if strings.HasPrefix(customID, "mysuggestions_prev_") {
			b.handleMySuggestionsPrev(s, i)
		} else if strings.HasPrefix(customID, "mysuggestions_next_") {
			b.handleMySuggestionsNext(s, i)
		}
	}
}