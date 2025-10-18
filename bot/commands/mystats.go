package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandleMyStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	count, avgRating, _ := h.db.GetUserStats(i.Member.User.ID)

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
