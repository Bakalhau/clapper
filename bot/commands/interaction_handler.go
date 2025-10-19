package commands

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		switch i.ApplicationCommandData().Name {
		case "setup":
			h.HandleSetup(s, i)
		case "config":
			h.HandleConfig(s, i)
		case "suggestion":
			h.HandleSuggestion(s, i)
		case "mystats":
			h.HandleMyStats(s, i)
		case "mysuggestions":
			h.HandleMySuggestions(s, i)
		case "suggestions":
			h.HandleSuggestions(s, i)
		case "pickmovie":
			h.HandlePickMovie(s, i)
		case "moviestats":
			h.HandleMovieStats(s, i)
		case "removesuggestion":
			h.HandleRemoveSuggestion(s, i)
		case "ratemovie":
			h.HandleRateMovie(s, i)
		case "moviereviews":
			h.HandleMovieReviews(s, i)
		case "selectedmovies":
			h.HandleSelectedMovies(s, i)
		}
	case discordgo.InteractionMessageComponent:
		customID := i.MessageComponentData().CustomID
		if strings.HasPrefix(customID, "reroll_movie_") {
			h.HandleRerollMovie(s, i)
		} else if strings.HasPrefix(customID, "confirm_movie_") {
			h.HandleConfirmMovie(s, i)
		} else if strings.HasPrefix(customID, "mysuggestions_prev_") {
			h.HandleMySuggestionsPrev(s, i)
		} else if strings.HasPrefix(customID, "mysuggestions_next_") {
			h.HandleMySuggestionsNext(s, i)
		} else if strings.HasPrefix(customID, "suggestions_prev_") {
			h.HandleSuggestionsPrev(s, i)
		} else if strings.HasPrefix(customID, "suggestions_next_") {
			h.HandleSuggestionsNext(s, i)
		}
	}
}