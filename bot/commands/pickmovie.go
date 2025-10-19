package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandlePickMovie(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	})

	movie, err := h.db.GetRandomMovie(guildID)
	if err != nil || movie == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ No available movies to pick! All suggestions have been selected or there are no suggestions yet."),
		})
		return
	}

	serverName := "Server"
	guild, err := s.Guild(guildID)
	if err == nil {
		serverName = guild.Name
	}

	tmdbMovie, _ := h.tmdb.GetMovieByID(movie.TMDBID)

	totalSuggestions, _ := h.db.GetAllSuggestionsCount(guildID)
	selectedCount, _ := h.db.GetSelectedMoviesCount(guildID)
	remaining := totalSuggestions - selectedCount

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎬 Movie Suggestion: %s (%s)", movie.MovieName, movie.ReleaseYear),
		Description: fmt.Sprintf("This movie has been randomly selected for %s!\n\nAdmins can reroll or confirm the selection.", serverName),
		Color:       0xFFD700,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "⭐ Rating", Value: fmt.Sprintf("%.1f/10", movie.Rating), Inline: true},
			{Name: "🎭 Genres", Value: movie.Genres, Inline: true},
			{Name: "📅 Year", Value: movie.ReleaseYear, Inline: true},
			{Name: "👤 Suggested by", Value: movie.Username, Inline: false},
			{Name: "📈 Progress", Value: fmt.Sprintf("%d/%d movies selected (%d remaining)", selectedCount, totalSuggestions, remaining), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Picked by %s", i.Member.User.Username),
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	if tmdbMovie != nil {
		posterURL := h.tmdb.GetPosterURL(tmdbMovie.PosterPath)
		if posterURL != "" {
			embed.Image = &discordgo.MessageEmbedImage{URL: posterURL}
		}
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Reroll",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("reroll_movie_%s_%d", guildID, movie.ID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔄",
					},
				},
				discordgo.Button{
					Label:    "Confirm Selection",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("confirm_movie_%s_%d", guildID, movie.ID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "✅",
					},
				},
			},
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
}

func (h *Handlers) HandleRerollMovie(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		return
	}

	isAdmin := false
	if i.Member != nil {
		perms, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
		if err == nil {
			isAdmin = perms&discordgo.PermissionAdministrator != 0
		}
	}

	if !isAdmin {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Only administrators can reroll movies!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	movie, err := h.db.GetRandomMovie(guildID)
	if err != nil || movie == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    ptrString("❌ No more available movies to pick!"),
			Embeds:     &[]*discordgo.MessageEmbed{},
			Components: &[]discordgo.MessageComponent{},
		})
		return
	}

	serverName := "Server"
	guild, err := s.Guild(guildID)
	if err == nil {
		serverName = guild.Name
	}

	tmdbMovie, _ := h.tmdb.GetMovieByID(movie.TMDBID)

	totalSuggestions, _ := h.db.GetAllSuggestionsCount(guildID)
	selectedCount, _ := h.db.GetSelectedMoviesCount(guildID)
	remaining := totalSuggestions - selectedCount

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎬 Movie Suggestion: %s (%s)", movie.MovieName, movie.ReleaseYear),
		Description: fmt.Sprintf("This movie has been randomly selected for %s!\n\nAdmins can reroll or confirm the selection.", serverName),
		Color:       0xFFD700,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "⭐ Rating", Value: fmt.Sprintf("%.1f/10", movie.Rating), Inline: true},
			{Name: "🎭 Genres", Value: movie.Genres, Inline: true},
			{Name: "📅 Year", Value: movie.ReleaseYear, Inline: true},
			{Name: "👤 Suggested by", Value: movie.Username, Inline: false},
			{Name: "📈 Progress", Value: fmt.Sprintf("%d/%d movies selected (%d remaining)", selectedCount, totalSuggestions, remaining), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Rerolled by %s", i.Member.User.Username),
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	if tmdbMovie != nil {
		posterURL := h.tmdb.GetPosterURL(tmdbMovie.PosterPath)
		if posterURL != "" {
			embed.Image = &discordgo.MessageEmbedImage{URL: posterURL}
		}
	}

	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Reroll",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("reroll_movie_%s_%d", guildID, movie.ID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔄",
					},
				},
				discordgo.Button{
					Label:    "Confirm Selection",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("confirm_movie_%s_%d", guildID, movie.ID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "✅",
					},
				},
			},
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
	})
}

func (h *Handlers) HandleConfirmMovie(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	if guildID == "" {
		return
	}

	isAdmin := false
	if i.Member != nil {
		perms, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
		if err == nil {
			isAdmin = perms&discordgo.PermissionAdministrator != 0
		}
	}

	if !isAdmin {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Only administrators can confirm movie selections!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	customID := i.MessageComponentData().CustomID
	re := regexp.MustCompile(`confirm_movie_([^_]+)_(\d+)`)
	matches := re.FindStringSubmatch(customID)

	if len(matches) < 3 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred. Please try again."),
		})
		return
	}

	extractedGuildID := matches[1]
	movieID, _ := strconv.Atoi(matches[2])

	// Verificação de segurança: guild do botão deve corresponder à guild atual
	if extractedGuildID != guildID {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ This selection is for a different server."),
		})
		return
	}

	if err := h.db.MarkMovieSelected(guildID, movieID); err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred while confirming the movie. Please try again."),
		})
		return
	}

	movie, err := h.db.GetMovieByID(movieID)
	if err != nil || movie == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ Could not find movie information."),
		})
		return
	}

	serverName := "Server"
	guild, err := s.Guild(guildID)
	if err == nil {
		serverName = guild.Name
	}

	totalSuggestions, _ := h.db.GetAllSuggestionsCount(guildID)
	selectedCount, _ := h.db.GetSelectedMoviesCount(guildID)
	remaining := totalSuggestions - selectedCount

	tmdbMovie, _ := h.tmdb.GetMovieByID(movie.TMDBID)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎉 Selected Movie: %s (%s)", movie.MovieName, movie.ReleaseYear),
		Description: fmt.Sprintf("This movie has been confirmed for %s!", serverName),
		Color:       0x00FF00,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "⭐ Rating", Value: fmt.Sprintf("%.1f/10", movie.Rating), Inline: true},
			{Name: "🎭 Genres", Value: movie.Genres, Inline: true},
			{Name: "📅 Year", Value: movie.ReleaseYear, Inline: true},
			{Name: "👤 Suggested by", Value: movie.Username, Inline: false},
			{Name: "📈 Progress", Value: fmt.Sprintf("%d/%d movies selected (%d remaining)", selectedCount, totalSuggestions, remaining), Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Confirmed by %s", i.Member.User.Username),
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	if tmdbMovie != nil {
		posterURL := h.tmdb.GetPosterURL(tmdbMovie.PosterPath)
		if posterURL != "" {
			embed.Image = &discordgo.MessageEmbedImage{URL: posterURL}
		}
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{},
	})
}