package bot



import (

	"clapper/bot/commands"

	"clapper/config"

	"clapper/database"

	"clapper/tmdb"

	"fmt"

	"log"



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



	commandHandlers := commands.NewHandlers(b.db, b.tmdb, b.config)



	b.session.AddHandler(commandHandlers.HandleInteraction)



	for _, cmd := range commands.Commands {

		_, err := b.session.ApplicationCommandCreate(b.session.State.User.ID, "", cmd)

		if err != nil {

			return fmt.Errorf("error creating command %s: %w", cmd.Name, err)

		}

		log.Printf("Registered command: %s", cmd.Name)

	}



	return nil

}



func (b *Bot) Stop() {

	b.session.Close()

	b.db.Close()

}
