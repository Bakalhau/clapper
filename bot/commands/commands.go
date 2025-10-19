package commands

import "github.com/bwmarrin/discordgo"

var Commands = []*discordgo.ApplicationCommand{
	{
		Name:        "setup",
		Description: "Configure the bot for this server (Admin only)",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionChannel,
				Name:        "suggestion_channel",
				Description: "The channel where movie suggestions will be posted",
				Required:    true,
				ChannelTypes: []discordgo.ChannelType{
					discordgo.ChannelTypeGuildText,
				},
			},
		},
	},
	{
		Name:        "config",
		Description: "View the current bot configuration for this server",
	},
	{
		Name:        "suggestion",
		Description: "Suggest a movie for your server using movie name or TMDB link",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "movie_name",
				Description: "Movie name or TMDB link (e.g., https://www.themoviedb.org/movie/74)",
				Required:    true,
			},
		},
	},
	{
		Name:        "mystats",
		Description: "View your movie suggestion statistics",
	},
	{
		Name:        "mysuggestions",
		Description: "View all your movie suggestions",
	},
	{
		Name:        "pickmovie",
		Description: "Pick a random movie from the suggestions",
	},
	{
		Name:        "moviestats",
		Description: "View overall movie suggestion statistics",
	},
	{
		Name:        "removesuggestion",
		Description: "Remove a movie suggestion",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "movie_name",
				Description: "The name of the movie to remove",
				Required:    true,
			},
		},
	},
	{
		Name:        "ratemovie",
		Description: "Rate a selected movie with a score and optional review",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "movie_name",
				Description: "The name of the movie to rate",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "rating",
				Description: "Your rating from 0 to 10 (e.g., 8.4)",
				Required:    true,
			},
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "review",
				Description: "Your review of the movie (optional)",
				Required:    false,
			},
		},
	},
	{
		Name:        "moviereviews",
		Description: "View all reviews for a selected movie",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionString,
				Name:        "movie_name",
				Description: "The name of the movie",
				Required:    true,
			},
		},
	},
	{
		Name:        "selectedmovies",
		Description: "View all movies that have been selected",
	},
}