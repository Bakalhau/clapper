package commands

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

func (h *Handlers) HandleSuggestion(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Check if server is configured
	guildConfig, err := h.db.GetGuildConfig(guildID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred while checking server configuration. Please try again."),
		})
		return
	}

	if guildConfig == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ This server has not been configured yet!\n\nAn administrator needs to run `/setup` to configure the suggestion channel before movies can be suggested."),
		})
		return
	}

	options := i.ApplicationCommandData().Options
	input := options[0].StringValue()

	tmdbID := extractTMDBID(input)

	var movie *tmdb.Movie

	if tmdbID > 0 {
		movie, err = h.tmdb.GetMovieByID(tmdbID)
		if err != nil || movie == nil {
			msg := fmt.Sprintf("❌ Could not find a movie with TMDB ID %d. Please check the link and try again.", tmdbID)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
			return
		}
	} else {
		movie, err = h.tmdb.SearchMovie(input)
		if err != nil || movie == nil {
			msg := fmt.Sprintf("❌ Could not find a movie named \"%s\". Please check the spelling and try again.", input)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
			return
		}
	}

	exists, _ := h.db.MovieAlreadySuggested(guildID, movie.ID)
	if exists {
		suggester, _ := h.db.GetMovieSuggester(guildID, movie.ID)
		msg := fmt.Sprintf("⚠️ **%s** has already been suggested by **%s** in this server!", movie.Title, suggester)
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

	posterURL := h.tmdb.GetPosterURL(movie.PosterPath)
	if posterURL != "" {
		embed.Image = &discordgo.MessageEmbedImage{URL: posterURL}
	}

	// Post to configured suggestion channel
	_, err = s.ChannelMessageSendEmbed(guildConfig.SuggestionChannelID, embed)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ Could not post to the suggestion channel. The channel may have been deleted or the bot may not have permissions. Please contact an administrator to run `/setup` again."),
		})
		return
	}

	_, err = h.db.SaveSuggestion(&database.Suggestion{
		GuildID:     guildID,
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

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Content: ptrString(fmt.Sprintf("✅ Successfully suggested **%s**! Your suggestion has been posted in <#%s>.", movie.Title, guildConfig.SuggestionChannelID)),
	})
}

func extractTMDBID(input string) int {
	re := regexp.MustCompile(`themoviedb.org/movie/(\d+)`)
	matches := re.FindStringSubmatch(input)

	if len(matches) > 1 {
		id, err := strconv.Atoi(matches[1])
		if err == nil {
			return id
		}
	}

	return 0
}