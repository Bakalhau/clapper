package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	BaseURL      = "https://api.themoviedb.org/3"
	ImageBaseURL = "https://image.tmdb.org/t/p/w500"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type SearchResponse struct {
	Results []Movie `json:"results"`
}

type Movie struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Overview    string  `json:"overview"`
	VoteAverage float64 `json:"vote_average"`
	ReleaseDate string  `json:"release_date"`
	PosterPath  string  `json:"poster_path"`
	GenreIDs    []int   `json:"genre_ids"`
}

var genreMap = map[int]string{
	28:    "Action",
	12:    "Adventure",
	16:    "Animation",
	35:    "Comedy",
	80:    "Crime",
	99:    "Documentary",
	18:    "Drama",
	10751: "Family",
	14:    "Fantasy",
	36:    "History",
	27:    "Horror",
	10402: "Music",
	9648:  "Mystery",
	10749: "Romance",
	878:   "Science Fiction",
	10770: "TV Movie",
	53:    "Thriller",
	10752: "War",
	37:    "Western",
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{},
	}
}

func (c *Client) SearchMovie(movieName string) (*Movie, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("query", movieName)
	params.Set("language", "en-US")

	url := fmt.Sprintf("%s/search/movie?%s", BaseURL, params.Encode())

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API returned status %d", resp.StatusCode)
	}

	var searchResp SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	if len(searchResp.Results) == 0 {
		return nil, nil
	}

	return &searchResp.Results[0], nil
}

// GetMovieByID busca um filme específico pelo ID do TMDB
func (c *Client) GetMovieByID(movieID int) (*Movie, error) {
	params := url.Values{}
	params.Set("api_key", c.apiKey)
	params.Set("language", "en-US")

	url := fmt.Sprintf("%s/movie/%d?%s", BaseURL, movieID, params.Encode())

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API returned status %d", resp.StatusCode)
	}

	var movie Movie
	if err := json.NewDecoder(resp.Body).Decode(&movie); err != nil {
		return nil, err
	}

	return &movie, nil
}

func (c *Client) GetPosterURL(posterPath string) string {
	if posterPath == "" {
		return ""
	}
	return ImageBaseURL + posterPath
}

func FormatGenres(genreIDs []int) string {
	var genres []string
	for _, id := range genreIDs {
		if name, ok := genreMap[id]; ok {
			genres = append(genres, name)
			if len(genres) >= 5 {
				break
			}
		}
	}

	if len(genres) == 0 {
		return "Not available"
	}

	result := genres[0]
	for i := 1; i < len(genres); i++ {
		result += ", " + genres[i]
	}
	return result
}