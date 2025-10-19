package commands

import (
	"clapper/database"
	"fmt"
	"regexp"
	"strconv"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandleSuggestions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ This command can only be used in a server.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	suggestions, err := h.db.GetAllSuggestions(guildID)
	if err != nil || len(suggestions) == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ No movies have been suggested yet in this server!"),
		})
		return
	}

	h.showAllSuggestionsPage(s, i, suggestions, 0)
}

func (h *Handlers) showAllSuggestionsPage(s *discordgo.Session, i *discordgo.InteractionCreate, suggestions []database.Suggestion, currentIndex int) {
	if currentIndex < 0 || currentIndex >= len(suggestions) {
		return
	}

	movie := suggestions[currentIndex]

	tmdbMovie, _ := h.tmdb.GetMovieByID(movie.TMDBID)

	statusEmoji := "⏳"
	statusText := "Not selected yet"
	embedColor := 0x3498DB

	if movie.IsSelected {
		statusEmoji = "✅"
		statusText = "Already selected"
		embedColor = 0x2ECC71
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s (%s)", statusEmoji, movie.MovieName, movie.ReleaseYear),
		Description: fmt.Sprintf("**Status:** %s\n**Suggested by:** %s\n**Suggested:** %s", statusText, movie.Username, movie.SuggestedAt.Format("Jan 02, 2006")),
		Color:       embedColor,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "⭐ Rating", Value: fmt.Sprintf("%.1f/10", movie.Rating), Inline: true},
			{Name: "🎭 Genres", Value: movie.Genres, Inline: true},
			{Name: "📅 Year", Value: movie.ReleaseYear, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d of %d", currentIndex+1, len(suggestions)),
		},
	}

	if tmdbMovie != nil {
		posterURL := h.tmdb.GetPosterURL(tmdbMovie.PosterPath)
		if posterURL != "" {
			embed.Image = &discordgo.MessageEmbedImage{URL: posterURL}
		}

		if tmdbMovie.Overview != "" {
			overview := tmdbMovie.Overview
			if len(overview) > 200 {
				overview = overview[:200] + "..."
			}
			embed.Description = fmt.Sprintf("%s\n\n%s", embed.Description, overview)
		}
	}

	components := []discordgo.MessageComponent{}

	buttons := []discordgo.MessageComponent{}

	prevButton := discordgo.Button{
		Label:    "Previous",
		Style:    discordgo.PrimaryButton,
		CustomID: fmt.Sprintf("suggestions_prev_%s_%d", i.GuildID, currentIndex),
		Emoji: &discordgo.ComponentEmoji{
			Name: "⬅️",
		},
		Disabled: currentIndex == 0,
	}

	nextButton := discordgo.Button{
		Label:    "Next",
		Style:    discordgo.PrimaryButton,
		CustomID: fmt.Sprintf("suggestions_next_%s_%d", i.GuildID, currentIndex),
		Emoji: &discordgo.ComponentEmoji{
			Name: "➡️",
		},
		Disabled: currentIndex == len(suggestions)-1,
	}

	buttons = append(buttons, prevButton, nextButton)

	if len(buttons) > 0 {
		components = append(components, discordgo.ActionsRow{
			Components: buttons,
		})
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
}

func (h *Handlers) HandleSuggestionsPrev(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	customID := i.MessageComponentData().CustomID
	re := regexp.MustCompile(`suggestions_prev_([^_]+)_(\d+)`)
	matches := re.FindStringSubmatch(customID)

	if len(matches) < 3 {
		return
	}

	currentIndex, _ := strconv.Atoi(matches[2])

	suggestions, err := h.db.GetAllSuggestions(guildID)
	if err != nil || len(suggestions) == 0 {
		return
	}

	newIndex := currentIndex - 1
	if newIndex < 0 {
		newIndex = 0
	}

	h.showAllSuggestionsPage(s, i, suggestions, newIndex)
}

func (h *Handlers) HandleSuggestionsNext(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	customID := i.MessageComponentData().CustomID
	re := regexp.MustCompile(`suggestions_next_([^_]+)_(\d+)`)
	matches := re.FindStringSubmatch(customID)

	if len(matches) < 3 {
		return
	}

	currentIndex, _ := strconv.Atoi(matches[2])

	suggestions, err := h.db.GetAllSuggestions(guildID)
	if err != nil || len(suggestions) == 0 {
		return
	}

	newIndex := currentIndex + 1
	if newIndex >= len(suggestions) {
		newIndex = len(suggestions) - 1
	}

	h.showAllSuggestionsPage(s, i, suggestions, newIndex)
}