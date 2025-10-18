package commands

import (
	"clapper/database"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandleRateMovie(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	options := i.ApplicationCommandData().Options
	movieName := options[0].StringValue()
	rating := options[1].FloatValue()
	
	var reviewText string
	if len(options) > 2 {
		reviewText = options[2].StringValue()
	}

	if rating < 0 || rating > 10 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ Invalid rating! Please provide a rating between 0 and 10."),
		})
		return
	}

	movie, err := h.db.SearchSelectedMovie(movieName)
	if err != nil || movie == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString(fmt.Sprintf("❌ Could not find a selected movie matching \"%s\".\nYou can only rate movies that have already been selected.", movieName)),
		})
		return
	}

	existingReview, _ := h.db.GetUserReview(movie.ID, i.Member.User.ID)

	review := &database.MovieReview{
		SuggestionID: movie.ID,
		UserID:       i.Member.User.ID,
		Username:     i.Member.User.Username,
		Rating:       rating,
		ReviewText:   reviewText,
	}

	if err := h.db.SaveMovieReview(review); err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred while saving your review. Please try again."),
		})
		return
	}

	avgRating, reviewCount, _ := h.db.GetAverageMovieRating(movie.ID)

	action := "added"
	if existingReview != nil {
		action = "updated"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("⭐ Review %s for %s", action, movie.MovieName),
		Color:       0xFFD700,
		Description: fmt.Sprintf("Your rating: **%.1f/10**", rating),
		Fields:      []*discordgo.MessageEmbedField{},
	}

	if reviewText != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📝 Your Review",
			Value:  reviewText,
			Inline: false,
		})
	}

	embed.Fields = append(embed.Fields,
		&discordgo.MessageEmbedField{
			Name:   "📊 Community Rating",
			Value:  fmt.Sprintf("%.1f/10 (%d review%s)", avgRating, reviewCount, pluralize(reviewCount)),
			Inline: true,
		},
		&discordgo.MessageEmbedField{
			Name:   "🎭 TMDB Rating",
			Value:  fmt.Sprintf("%.1f/10", movie.Rating),
			Inline: true,
		},
	)

	tmdbMovie, _ := h.tmdb.GetMovieByID(movie.TMDBID)
	if tmdbMovie != nil {
		posterURL := h.tmdb.GetPosterURL(tmdbMovie.PosterPath)
		if posterURL != "" {
			embed.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: posterURL}
		}
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}