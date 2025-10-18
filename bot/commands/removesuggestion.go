package commands

import (
	"clapper/database"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandleRemoveSuggestion(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	options := i.ApplicationCommandData().Options
	movieName := options[0].StringValue()

	isAdmin := false
	if i.Member != nil {
		perms, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
		if err == nil {
			isAdmin = perms&discordgo.PermissionAdministrator != 0
		}
	}

	var movie *database.Suggestion
	var err error

	if isAdmin {
		movie, err = h.db.SearchAnySuggestion(movieName)
	} else {
		movie, err = h.db.SearchUserSuggestions(movieName, i.Member.User.ID)
	}

	if err != nil || movie == nil {
		msg := fmt.Sprintf("❌ Could not find a movie suggestion matching \"%s\".", movieName)
		if !isAdmin {
			msg = fmt.Sprintf("❌ Could not find a movie named \"%s\" in your suggestions.\nYou can only remove movies that you suggested.", movieName)
		}
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: ptrString(msg)})
		return
	}

	if err := h.db.RemoveSuggestion(movie.ID); err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred while removing the movie. Please try again."),
		})
		return
	}

	suggesterInfo := ""
	if isAdmin && movie.UserID != i.Member.User.ID {
		suggesterInfo = fmt.Sprintf(" (suggested by %s)", movie.Username)
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: ptrString(fmt.Sprintf("✅ Successfully removed **%s**%s from suggestions.", movie.MovieName, suggesterInfo)),
	})
}
