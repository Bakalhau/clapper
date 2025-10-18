package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandleMovieReviews(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	options := i.ApplicationCommandData().Options
	movieName := options[0].StringValue()

	movie, err := h.db.SearchSelectedMovie(movieName)
	if err != nil || movie == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString(fmt.Sprintf("❌ Could not find a selected movie matching \"%s\".", movieName)),
		})
		return
	}

	reviews, err := h.db.GetMovieReviews(movie.ID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred while fetching reviews. Please try again."),
		})
		return
	}

	avgRating, reviewCount, _ := h.db.GetAverageMovieRating(movie.ID)

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🎬 Reviews for %s (%s)", movie.MovieName, movie.ReleaseYear),
		Color:       0x9B59B6,
		Description: "",
		Fields:      []*discordgo.MessageEmbedField{},
	}

	if reviewCount > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📊 Community Rating",
			Value:  fmt.Sprintf("⭐ **%.1f/10** based on %d review%s", avgRating, reviewCount, pluralize(reviewCount)),
			Inline: false,
		})
	} else {
		embed.Description = "No reviews yet. Be the first to rate this movie with `/ratemovie`!"
	}

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "🎭 TMDB Rating",
		Value:  fmt.Sprintf("%.1f/10", movie.Rating),
		Inline: true,
	})

	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "🎭 Genres",
		Value:  movie.Genres,
		Inline: true,
	})

	if len(reviews) > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "━━━━━━━━━━━━━━━━━━━━",
			Value:  "**User Reviews**",
			Inline: false,
		})

		maxReviews := 5
		if len(reviews) < maxReviews {
			maxReviews = len(reviews)
		}

		for i := 0; i < maxReviews; i++ {
			review := reviews[i]
			
			value := fmt.Sprintf("⭐ **%.1f/10**", review.Rating)
			
			if review.ReviewText != "" {
				reviewText := review.ReviewText
				if len(reviewText) > 200 {
					reviewText = reviewText[:200] + "..."
				}
				value += fmt.Sprintf("\n*\"%s\"*", reviewText)
			}
			
			value += fmt.Sprintf("\n— %s", review.Username)

			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("Review #%d", i+1),
				Value:  value,
				Inline: false,
			})
		}

		if len(reviews) > maxReviews {
			embed.Footer = &discordgo.MessageEmbedFooter{
				Text: fmt.Sprintf("Showing %d of %d reviews", maxReviews, len(reviews)),
			}
		}
	}

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

func (h *Handlers) HandleSelectedMovies(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})

	movies, err := h.db.GetAllSelectedMovies()
	if err != nil || len(movies) == 0 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ No movies have been selected yet!"),
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎬 Selected Movies",
		Color:       0x2ECC71,
		Description: fmt.Sprintf("Total movies selected: **%d**\n━━━━━━━━━━━━━━━━━━━━", len(movies)),
		Fields:      []*discordgo.MessageEmbedField{},
	}

	maxMovies := 10
	if len(movies) < maxMovies {
		maxMovies = len(movies)
	}

	for i := 0; i < maxMovies; i++ {
		movie := movies[i]
		
		var value strings.Builder
		value.WriteString(fmt.Sprintf("**%s** (%s)\n", movie.MovieName, movie.ReleaseYear))
		value.WriteString(fmt.Sprintf("🎭 TMDB: %.1f/10\n", movie.Rating))
		
		if movie.ReviewCount > 0 {
			value.WriteString(fmt.Sprintf("⭐ Community: %.1f/10 (%d review%s)\n", 
				movie.AverageScore, movie.ReviewCount, pluralize(movie.ReviewCount)))
		} else {
			value.WriteString("⭐ Community: No reviews yet\n")
		}
		
		value.WriteString(fmt.Sprintf("👤 Suggested by: %s", movie.Username))

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("#%d", i+1),
			Value:  value.String(),
			Inline: false,
		})
	}

	if len(movies) > maxMovies {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Showing %d of %d movies", maxMovies, len(movies)),
		}
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}