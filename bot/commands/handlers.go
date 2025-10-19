package commands

import (
	"clapper/database"
	"clapper/tmdb"
)

type Handlers struct {
	db   *database.Database
	tmdb *tmdb.Client
}

func NewHandlers(db *database.Database, tmdb *tmdb.Client) *Handlers {
	return &Handlers{
		db:   db,
		tmdb: tmdb,
	}
}