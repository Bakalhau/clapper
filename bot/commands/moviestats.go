package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandleMovieStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	totalSuggestions, _ := h.db.GetAllSuggestionsCount(guildID)
	selectedCount, _ := h.db.GetSelectedMoviesCount(guildID)
	remaining := totalSuggestions - selectedCount

	serverName := "Server"
	guild, err := s.Guild(guildID)
	if err == nil {
		serverName = guild.Name
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