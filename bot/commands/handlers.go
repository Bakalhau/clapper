package commands

import (
	"clapper/config"
	"clapper/database"
	"clapper/tmdb"
)

type Handlers struct {
	db      *database.Database
	tmdb    *tmdb.Client
	config  *config.Config
}

func NewHandlers(db *database.Database, tmdb *tmdb.Client, config *config.Config) *Handlers {
	return &Handlers{
		db:      db,
		tmdb:    tmdb,
		config:  config,
	}
}
