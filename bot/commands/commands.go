package commands

import "github.com/bwmarrin/discordgo"

var Commands = []*discordgo.ApplicationCommand{
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
}
