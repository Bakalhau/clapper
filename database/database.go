package database

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

type Suggestion struct {
	ID           int
	GuildID      string
	MovieName    string
	UserID       string
	Username     string
	SuggestedAt  time.Time
	TMDBID       int
	Rating       float64
	Genres       string
	ReleaseYear  string
	IsSelected   bool
}

type MovieResult struct {
	ID          int
	GuildID     string
	MovieName   string
	Username    string
	Rating      float64
	Genres      string
	ReleaseYear string
	TMDBID      int
}

type MovieReview struct {
	ID           int
	SuggestionID int
	GuildID      string
	UserID       string
	Username     string
	Rating       float64
	ReviewText   string
	ReviewedAt   time.Time
}

type SelectedMovieWithReviews struct {
	MovieResult
	Reviews      []MovieReview
	AverageScore float64
	ReviewCount  int
}

func New(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	d := &Database{db: db}
	if err := d.initDatabase(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *Database) initDatabase() error {
	// Verificar se precisa migrar
	var hasGuildID bool
	err := d.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('suggestions') WHERE name='guild_id'").Scan(&hasGuildID)
	
	if err == nil && !hasGuildID {
		log.Println("Migrando banco de dados para suporte multi-guild...")
		
		// Backup das tabelas antigas
		_, _ = d.db.Exec("ALTER TABLE suggestions RENAME TO suggestions_old")
		_, _ = d.db.Exec("ALTER TABLE selected_movies RENAME TO selected_movies_old")
		_, _ = d.db.Exec("ALTER TABLE movie_reviews RENAME TO movie_reviews_old")
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS suggestions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT NOT NULL,
			movie_name TEXT NOT NULL,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			suggested_at TIMESTAMP NOT NULL,
			tmdb_id INTEGER NOT NULL,
			rating REAL,
			genres TEXT,
			release_year TEXT,
			UNIQUE(guild_id, tmdb_id)
		)`,
		`CREATE TABLE IF NOT EXISTS selected_movies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			guild_id TEXT NOT NULL,
			suggestion_id INTEGER NOT NULL,
			selected_at TIMESTAMP NOT NULL,
			FOREIGN KEY (suggestion_id) REFERENCES suggestions (id) ON DELETE CASCADE,
			UNIQUE(guild_id, suggestion_id)
		)`,
		`CREATE TABLE IF NOT EXISTS movie_reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			suggestion_id INTEGER NOT NULL,
			guild_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			rating REAL NOT NULL CHECK(rating >= 0 AND rating <= 10),
			review_text TEXT,
			reviewed_at TIMESTAMP NOT NULL,
			FOREIGN KEY (suggestion_id) REFERENCES suggestions (id) ON DELETE CASCADE,
			UNIQUE(guild_id, suggestion_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_suggestions_guild_id ON suggestions(guild_id)`,
		`CREATE INDEX IF NOT EXISTS idx_suggestions_user_id ON suggestions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_suggestions_tmdb_id ON suggestions(tmdb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_suggestions_guild_tmdb ON suggestions(guild_id, tmdb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_selected_movies_guild_id ON selected_movies(guild_id)`,
		`CREATE INDEX IF NOT EXISTS idx_selected_movies_suggestion_id ON selected_movies(suggestion_id)`,
		`CREATE INDEX IF NOT EXISTS idx_movie_reviews_guild_id ON movie_reviews(guild_id)`,
		`CREATE INDEX IF NOT EXISTS idx_movie_reviews_suggestion_id ON movie_reviews(suggestion_id)`,
		`CREATE INDEX IF NOT EXISTS idx_movie_reviews_user_id ON movie_reviews(user_id)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			log.Printf("Erro ao executar query: %s\nErro: %v", query, err)
			return err
		}
	}

	log.Println("Database initialized successfully with multi-guild support")
	return nil
}

func (d *Database) MovieAlreadySuggested(guildID string, tmdbID int) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM suggestions WHERE guild_id = ? AND tmdb_id = ?", guildID, tmdbID).Scan(&count)
	return count > 0, err
}

func (d *Database) GetMovieSuggester(guildID string, tmdbID int) (string, error) {
	var username string
	err := d.db.QueryRow("SELECT username FROM suggestions WHERE guild_id = ? AND tmdb_id = ?", guildID, tmdbID).Scan(&username)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return username, err
}

func (d *Database) SaveSuggestion(s *Suggestion) (int64, error) {
	result, err := d.db.Exec(`
		INSERT INTO suggestions (guild_id, movie_name, user_id, username, suggested_at, tmdb_id, rating, genres, release_year)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.GuildID, s.MovieName, s.UserID, s.Username, time.Now(), s.TMDBID, s.Rating, s.Genres, s.ReleaseYear,
	)
	if err != nil {
		log.Printf("Erro ao salvar sugestão: %v", err)
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	log.Printf("Sugestão salva com ID: %d para guild: %s", id, s.GuildID)
	return id, nil
}

func (d *Database) GetUserStats(guildID, userID string) (count int, avgRating float64, err error) {
	err = d.db.QueryRow(`
		SELECT COUNT(*), COALESCE(AVG(rating), 0)
		FROM suggestions
		WHERE guild_id = ? AND user_id = ?`, guildID, userID).Scan(&count, &avgRating)
	return
}

func (d *Database) GetUserSuggestions(guildID, userID string) ([]Suggestion, error) {
	log.Printf("Buscando sugestões para guild_id: %s, user_id: %s", guildID, userID)
	
	rows, err := d.db.Query(`
		SELECT s.id, s.guild_id, s.movie_name, s.user_id, s.username, s.suggested_at, 
		       s.tmdb_id, s.rating, s.genres, s.release_year,
		       CASE WHEN sm.id IS NOT NULL THEN 1 ELSE 0 END as is_selected
		FROM suggestions s
		LEFT JOIN selected_movies sm ON s.id = sm.suggestion_id AND s.guild_id = sm.guild_id
		WHERE s.guild_id = ? AND s.user_id = ?
		ORDER BY s.suggested_at DESC`, guildID, userID)
	
	if err != nil {
		log.Printf("Erro na query GetUserSuggestions: %v", err)
		return nil, err
	}
	defer rows.Close()

	var suggestions []Suggestion
	for rows.Next() {
		var s Suggestion
		var isSelectedInt int
		err := rows.Scan(&s.ID, &s.GuildID, &s.MovieName, &s.UserID, &s.Username, &s.SuggestedAt,
			&s.TMDBID, &s.Rating, &s.Genres, &s.ReleaseYear, &isSelectedInt)
		if err != nil {
			log.Printf("Erro ao escanear linha: %v", err)
			return nil, err
		}
		s.IsSelected = isSelectedInt == 1
		suggestions = append(suggestions, s)
	}

	log.Printf("Total de sugestões encontradas: %d", len(suggestions))
	return suggestions, nil
}

func (d *Database) GetRandomMovie(guildID string) (*MovieResult, error) {
	var m MovieResult
	err := d.db.QueryRow(`
		SELECT s.id, s.guild_id, s.movie_name, s.username, s.rating, s.genres, s.release_year, s.tmdb_id
		FROM suggestions s
		LEFT JOIN selected_movies sm ON s.id = sm.suggestion_id AND s.guild_id = sm.guild_id
		WHERE s.guild_id = ? AND sm.id IS NULL
		ORDER BY RANDOM()
		LIMIT 1`, guildID).Scan(&m.ID, &m.GuildID, &m.MovieName, &m.Username, &m.Rating, &m.Genres, &m.ReleaseYear, &m.TMDBID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (d *Database) GetMovieByID(suggestionID int) (*MovieResult, error) {
	var m MovieResult
	err := d.db.QueryRow(`
		SELECT id, guild_id, movie_name, username, rating, genres, release_year, tmdb_id
		FROM suggestions
		WHERE id = ?`, suggestionID).Scan(&m.ID, &m.GuildID, &m.MovieName, &m.Username, &m.Rating, &m.Genres, &m.ReleaseYear, &m.TMDBID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (d *Database) MarkMovieSelected(guildID string, suggestionID int) error {
	_, err := d.db.Exec(`
		INSERT INTO selected_movies (guild_id, suggestion_id, selected_at)
		VALUES (?, ?, ?)`, guildID, suggestionID, time.Now())
	return err
}

func (d *Database) GetAllSuggestionsCount(guildID string) (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM suggestions WHERE guild_id = ?", guildID).Scan(&count)
	return count, err
}

func (d *Database) GetSelectedMoviesCount(guildID string) (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM selected_movies WHERE guild_id = ?", guildID).Scan(&count)
	return count, err
}

func (d *Database) SearchUserSuggestions(guildID, movieName, userID string) (*Suggestion, error) {
	var s Suggestion
	err := d.db.QueryRow(`
		SELECT id, guild_id, movie_name, tmdb_id, user_id, username
		FROM suggestions
		WHERE guild_id = ? AND LOWER(movie_name) LIKE LOWER(?) AND user_id = ?
		LIMIT 1`, guildID, "%"+movieName+"%", userID).Scan(&s.ID, &s.GuildID, &s.MovieName, &s.TMDBID, &s.UserID, &s.Username)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Database) SearchAnySuggestion(guildID, movieName string) (*Suggestion, error) {
	var s Suggestion
	err := d.db.QueryRow(`
		SELECT id, guild_id, movie_name, tmdb_id, user_id, username
		FROM suggestions
		WHERE guild_id = ? AND LOWER(movie_name) LIKE LOWER(?)
		LIMIT 1`, guildID, "%"+movieName+"%").Scan(&s.ID, &s.GuildID, &s.MovieName, &s.TMDBID, &s.UserID, &s.Username)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Database) RemoveSuggestion(suggestionID int) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM movie_reviews WHERE suggestion_id = ?", suggestionID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM selected_movies WHERE suggestion_id = ?", suggestionID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM suggestions WHERE id = ?", suggestionID); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) IsMovieSelected(guildID string, suggestionID int) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM selected_movies WHERE guild_id = ? AND suggestion_id = ?", guildID, suggestionID).Scan(&count)
	return count > 0, err
}

func (d *Database) SaveMovieReview(review *MovieReview) error {
	_, err := d.db.Exec(`
		INSERT INTO movie_reviews (suggestion_id, guild_id, user_id, username, rating, review_text, reviewed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(guild_id, suggestion_id, user_id) 
		DO UPDATE SET rating = excluded.rating, review_text = excluded.review_text, reviewed_at = excluded.reviewed_at`,
		review.SuggestionID, review.GuildID, review.UserID, review.Username, review.Rating, review.ReviewText, time.Now())
	return err
}

func (d *Database) GetMovieReviews(guildID string, suggestionID int) ([]MovieReview, error) {
	rows, err := d.db.Query(`
		SELECT id, suggestion_id, guild_id, user_id, username, rating, COALESCE(review_text, ''), reviewed_at
		FROM movie_reviews
		WHERE guild_id = ? AND suggestion_id = ?
		ORDER BY reviewed_at DESC`, guildID, suggestionID)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []MovieReview
	for rows.Next() {
		var r MovieReview
		err := rows.Scan(&r.ID, &r.SuggestionID, &r.GuildID, &r.UserID, &r.Username, &r.Rating, &r.ReviewText, &r.ReviewedAt)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}

	return reviews, nil
}

func (d *Database) GetUserReview(guildID string, suggestionID int, userID string) (*MovieReview, error) {
	var r MovieReview
	err := d.db.QueryRow(`
		SELECT id, suggestion_id, guild_id, user_id, username, rating, COALESCE(review_text, ''), reviewed_at
		FROM movie_reviews
		WHERE guild_id = ? AND suggestion_id = ? AND user_id = ?`, guildID, suggestionID, userID).Scan(
		&r.ID, &r.SuggestionID, &r.GuildID, &r.UserID, &r.Username, &r.Rating, &r.ReviewText, &r.ReviewedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *Database) GetAverageMovieRating(guildID string, suggestionID int) (float64, int, error) {
	var avg sql.NullFloat64
	var count int
	err := d.db.QueryRow(`
		SELECT AVG(rating), COUNT(*)
		FROM movie_reviews
		WHERE guild_id = ? AND suggestion_id = ?`, guildID, suggestionID).Scan(&avg, &count)
	
	if err != nil {
		return 0, 0, err
	}
	
	if !avg.Valid {
		return 0, 0, nil
	}
	
	return avg.Float64, count, nil
}

func (d *Database) GetAllSelectedMovies(guildID string) ([]SelectedMovieWithReviews, error) {
	rows, err := d.db.Query(`
		SELECT s.id, s.guild_id, s.movie_name, s.username, s.rating, s.genres, s.release_year, s.tmdb_id
		FROM suggestions s
		INNER JOIN selected_movies sm ON s.id = sm.suggestion_id AND s.guild_id = sm.guild_id
		WHERE s.guild_id = ?
		ORDER BY sm.selected_at DESC`, guildID)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []SelectedMovieWithReviews
	for rows.Next() {
		var m SelectedMovieWithReviews
		err := rows.Scan(&m.ID, &m.GuildID, &m.MovieName, &m.Username, &m.Rating, &m.Genres, &m.ReleaseYear, &m.TMDBID)
		if err != nil {
			return nil, err
		}
		
		reviews, _ := d.GetMovieReviews(guildID, m.ID)
		m.Reviews = reviews
		m.ReviewCount = len(reviews)
		
		avgRating, _, _ := d.GetAverageMovieRating(guildID, m.ID)
		m.AverageScore = avgRating
		
		movies = append(movies, m)
	}

	return movies, nil
}

func (d *Database) SearchSelectedMovie(guildID, movieName string) (*MovieResult, error) {
	var m MovieResult
	err := d.db.QueryRow(`
		SELECT s.id, s.guild_id, s.movie_name, s.username, s.rating, s.genres, s.release_year, s.tmdb_id
		FROM suggestions s
		INNER JOIN selected_movies sm ON s.id = sm.suggestion_id AND s.guild_id = sm.guild_id
		WHERE s.guild_id = ? AND LOWER(s.movie_name) LIKE LOWER(?)
		LIMIT 1`, guildID, "%"+movieName+"%").Scan(&m.ID, &m.GuildID, &m.MovieName, &m.Username, &m.Rating, &m.Genres, &m.ReleaseYear, &m.TMDBID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}