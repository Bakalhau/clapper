package bot

import (
	"clapper/database"
	"clapper/tmdb"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (b *Bot) handleSuggestion(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	options := i.ApplicationCommandData().Options
	input := options[0].StringValue()

	tmdbID := extractTMDBID(input)
	
	var movie *tmdb.Movie
	var err error
	
	if tmdbID > 0 {
		movie, err = b.tmdb.GetMovieByID(tmdbID)
		if err != nil || movie == nil {
			msg := fmt.Sprintf("❌ Could not find a movie with TMDB ID %d. Please check the link and try again.", tmdbID)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
			return
		}
	} else {
		movie, err = b.tmdb.SearchMovie(input)
		if err != nil || movie == nil {
			msg := fmt.Sprintf("❌ Could not find a movie named \"%s\". Please check the spelling and try again.", input)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
			return
		}
	}

	exists, _ := b.db.MovieAlreadySuggested(movie.ID)
	if exists {
		suggester, _ := b.db.GetMovieSuggester(movie.ID)
		msg := fmt.Sprintf("⚠️ **%s** has already been suggested by **%s**!", movie.Title, suggester)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})
		return
	}

	year := "Unknown"
	if movie.ReleaseDate != "" {
		parts := strings.Split(movie.ReleaseDate, "-")
		if len(parts) > 0 {
			year = parts[0]
		}
	}

	genres := tmdb.FormatGenres(movie.GenreIDs)

	_, err = b.db.SaveSuggestion(&database.Suggestion{
		MovieName:   movie.Title,
		UserID:      i.Member.User.ID,
		Username:    i.Member.User.Username,
		TMDBID:      movie.ID,
		Rating:      movie.VoteAverage,
		Genres:      genres,
		ReleaseYear: year,
	})

	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred while saving your suggestion. Please try again later."),
		})
		return
	}

	overview := movie.Overview
	if len(overview) > 300 {
		overview = overview[:300] + "..."
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎬 %s (%s)", movie.Title, year),
		Description: overview,
		Color:       0xFFD700,
		Timestamp:   time.Now().Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "⭐ Rating", Value: fmt.Sprintf("%.1f/10", movie.VoteAverage), Inline: true},
			{Name: "🎭 Genres", Value: genres, Inline: true},
			{Name: "📅 Release Year", Value: year, Inline: true},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text:    fmt.Sprintf("Suggested by %s", i.Member.User.Username),
			IconURL: i.Member.User.AvatarURL(""),
		},
	}

	posterURL := b.tmdb.GetPosterURL(movie.PosterPath)
	if posterURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: posterURL}
	}

	_, err = s.ChannelMessageSendEmbed(b.config.SuggestionChannelID, embed)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ Could not find the suggestion channel. Please contact an administrator."),
		})
		return
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: ptrString(fmt.Sprintf("✅ Successfully suggested **%s**! Your suggestion has been posted in the movie channel.", movie.Title)),
	})
}

func extractTMDBID(input string) int {
	re := regexp.MustCompile(`themoviedb\.org/movie/(\d+)`)
	matches := re.FindStringSubmatch(input)
	
	if len(matches) > 1 {
		id, err := strconv.Atoi(matches[1])
		if err == nil {
			return id
		}
	}
	
	return 0
}

func (b *Bot) handleMyStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	count, avgRating, _ := b.db.GetUserStats(i.Member.User.ID)

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 %s's Statistics", i.Member.User.Username),
		Color: 0x0000FF,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Total Suggestions", Value: fmt.Sprintf("%d", count), Inline: true},
			{Name: "Average Rating", Value: fmt.Sprintf("%.1f/10", avgRating), Inline: true},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

func (b *Bot) handlePickMovie(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	movie, err := b.db.GetRandomMovie()
	if err != nil || movie == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ No available movies to pick! All suggestions have been selected or there are no suggestions yet."),
		})
		return
	}

	serverName := "Server"
	if i.GuildID != "" {
		guild, err := s.Guild(i.GuildID)
		if err == nil {
			serverName = guild.Name
		}
	}

	tmdbMovie, _ := b.tmdb.GetMovieByID(movie.TMDBID)
	
	totalSuggestions, _ := b.db.GetAllSuggestionsCount()
	selectedCount, _ := b.db.GetSelectedMoviesCount()
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
		posterURL := b.tmdb.GetPosterURL(tmdbMovie.PosterPath)
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
					CustomID: fmt.Sprintf("reroll_movie_%d", movie.ID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔄",
					},
				},
				discordgo.Button{
					Label:    "Confirm Selection",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("confirm_movie_%d", movie.ID),
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

func (b *Bot) handleMovieStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	totalSuggestions, _ := b.db.GetAllSuggestionsCount()
	selectedCount, _ := b.db.GetSelectedMoviesCount()
	remaining := totalSuggestions - selectedCount

	serverName := "Server"
	if i.GuildID != "" {
		guild, err := s.Guild(i.GuildID)
		if err == nil {
			serverName = guild.Name
		}
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("📊 %s Movies Statistics", serverName),
		Color: 0x800080,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Total Suggestions", Value: fmt.Sprintf("%d", totalSuggestions), Inline: true},
			{Name: "Movies Selected", Value: fmt.Sprintf("%d", selectedCount), Inline: true},
			{Name: "Movies Remaining", Value: fmt.Sprintf("%d", remaining), Inline: true},
		},
	}

	if totalSuggestions > 0 {
		percentage := (float64(selectedCount) / float64(totalSuggestions)) * 100
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Progress",
			Value:  fmt.Sprintf("%.1f%% of suggestions have been selected", percentage),
			Inline: false,
		})
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{embed},
		},
	})
}

func (b *Bot) handleRemoveSuggestion(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
		movie, err = b.db.SearchAnySuggestion(movieName)
	} else {
		movie, err = b.db.SearchUserSuggestions(movieName, i.Member.User.ID)
	}

	if err != nil || movie == nil {
		msg := fmt.Sprintf("❌ Could not find a movie suggestion matching \"%s\".", movieName)
		if !isAdmin {
			msg = fmt.Sprintf("❌ Could not find a movie named \"%s\" in your suggestions.\nYou can only remove movies that you suggested.", movieName)
		}
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: ptrString(msg)})
		return
	}

	if err := b.db.RemoveSuggestion(movie.ID); err != nil {
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

func ptrString(s string) *string {
	return &s
}

func (b *Bot) handleRerollMovie(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	movie, err := b.db.GetRandomMovie()
	if err != nil || movie == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content:    ptrString("❌ No more available movies to pick!"),
			Embeds:     &[]*discordgo.MessageEmbed{},
			Components: &[]discordgo.MessageComponent{},
		})
		return
	}

	serverName := "Server"
	if i.GuildID != "" {
		guild, err := s.Guild(i.GuildID)
		if err == nil {
			serverName = guild.Name
		}
	}

	tmdbMovie, _ := b.tmdb.GetMovieByID(movie.TMDBID)
	
	totalSuggestions, _ := b.db.GetAllSuggestionsCount()
	selectedCount, _ := b.db.GetSelectedMoviesCount()
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
		posterURL := b.tmdb.GetPosterURL(tmdbMovie.PosterPath)
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
					CustomID: fmt.Sprintf("reroll_movie_%d", movie.ID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔄",
					},
				},
				discordgo.Button{
					Label:    "Confirm Selection",
					Style:    discordgo.SuccessButton,
					CustomID: fmt.Sprintf("confirm_movie_%d", movie.ID),
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

func (b *Bot) handleConfirmMovie(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	re := regexp.MustCompile(`confirm_movie_(\d+)`)
	matches := re.FindStringSubmatch(customID)
	
	if len(matches) < 2 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred. Please try again."),
		})
		return
	}

	movieID, _ := strconv.Atoi(matches[1])

	if err := b.db.MarkMovieSelected(movieID); err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred while confirming the movie. Please try again."),
		})
		return
	}

	movie, err := b.db.GetMovieByID(movieID)
	if err != nil || movie == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ Could not find movie information."),
		})
		return
	}

	serverName := "Server"
	if i.GuildID != "" {
		guild, err := s.Guild(i.GuildID)
		if err == nil {
			serverName = guild.Name
		}
	}

	totalSuggestions, _ := b.db.GetAllSuggestionsCount()
	selectedCount, _ := b.db.GetSelectedMoviesCount()
	remaining := totalSuggestions - selectedCount

	tmdbMovie, _ := b.tmdb.GetMovieByID(movie.TMDBID)

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
		posterURL := b.tmdb.GetPosterURL(tmdbMovie.PosterPath)
		if posterURL != "" {
			embed.Image = &discordgo.MessageEmbedImage{URL: posterURL}
		}
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &[]discordgo.MessageComponent{},
	})
}