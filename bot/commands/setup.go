package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func (h *Handlers) HandleSetup(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	// Check if user is admin
	if i.Member == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ An error occurred. Please try again.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	perms, err := s.UserChannelPermissions(i.Member.User.ID, i.ChannelID)
	if err != nil || perms&discordgo.PermissionAdministrator == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Only administrators can configure the bot!",
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
	channelID := options[0].ChannelValue(s).ID

	// Verify if channel exists and bot has permissions
	channel, err := s.Channel(channelID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ Could not access the specified channel. Please make sure the bot has permission to view and send messages in that channel."),
		})
		return
	}

	// Check if it's a text channel
	if channel.Type != discordgo.ChannelTypeGuildText {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ Please select a text channel for movie suggestions."),
		})
		return
	}

	// Save configuration
	if err := h.db.SaveGuildConfig(guildID, channelID); err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred while saving the configuration. Please try again."),
		})
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Bot Configuration Updated",
		Description: fmt.Sprintf("Movie suggestions will now be posted in <#%s>", channelID),
		Color:       0x2ECC71,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📝 What's Next?",
				Value:  "Users can now start suggesting movies using `/suggestion`!\n\nAll suggestions will be automatically posted in the configured channel.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Configured by %s", i.Member.User.Username),
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}

func (h *Handlers) HandleConfig(s *discordgo.Session, i *discordgo.InteractionCreate) {
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

	config, err := h.db.GetGuildConfig(guildID)
	if err != nil {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: ptrString("❌ An error occurred while fetching the configuration."),
		})
		return
	}

	if config == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "⚙️ Bot Configuration",
			Description: "This server has not been configured yet.",
			Color:       0xE74C3C,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "🛠️ Setup Required",
					Value:  "An administrator needs to run `/setup` to configure the suggestion channel before users can suggest movies.",
					Inline: false,
				},
			},
		}
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &[]*discordgo.MessageEmbed{embed},
		})
		return
	}

	serverName := "Server"
	guild, err := s.Guild(guildID)
	if err == nil {
		serverName = guild.Name
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("⚙️ %s Configuration", serverName),
		Color: 0x3498DB,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📢 Suggestion Channel",
				Value:  fmt.Sprintf("<#%s>", config.SuggestionChannelID),
				Inline: false,
			},
			{
				Name:   "📅 Configured At",
				Value:  config.ConfiguredAt.Format("Jan 02, 2006 at 3:04 PM"),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Administrators can update this with /setup",
		},
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
}
