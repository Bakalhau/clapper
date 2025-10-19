package commands

import (
	"clapper/database"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandleRateMovie(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	options := i.ApplicationCommandData().Options
	movieName := options[0].StringValue()
	ratingStr := options[1].StringValue()
	
	var reviewText string
	if len(options) > 2 {
		reviewText = options[2].StringValue()
	}

	// Converter e validar o rating
	ratingStr = strings.TrimSpace(ratingStr)
	ratingStr = strings.Replace(ratingStr, ",", ".", -1)
	rating, err := strconv.ParseFloat(ratingStr, 64)
	
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ Invalid rating! Please provide a valid number (e.g., 8.4 or 9)."),
		})
		return
	}

	// Validação da nota
	if rating < 0 || rating > 10 {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ Invalid rating! Please provide a rating between 0 and 10."),
		})
		return
	}

	// Buscar o filme selecionado
	movie, err := h.db.SearchSelectedMovie(guildID, movieName)
	if err != nil || movie == nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString(fmt.Sprintf("❌ Could not find a selected movie matching \"%s\".\nYou can only rate movies that have already been selected.", movieName)),
		})
		return
	}

	// Verificar se o usuário já avaliou
	existingReview, _ := h.db.GetUserReview(guildID, movie.ID, i.Member.User.ID)

	// Salvar ou atualizar a avaliação
	review := &database.MovieReview{
		SuggestionID: movie.ID,
		GuildID:      guildID,
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

	// Buscar média e contagem de avaliações
	avgRating, reviewCount, _ := h.db.GetAverageMovieRating(guildID, movie.ID)

	// Criar embed de confirmação
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

	// Adicionar estatísticas do filme
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

	// Buscar poster do TMDB
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