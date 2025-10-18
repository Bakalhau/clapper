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
	var tableName string
	err := d.db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='selected_movies'").Scan(&tableName)
	
	if err == sql.ErrNoRows {
		log.Println("Criando tabela selected_movies...")
	} else if err == nil {
		var hasColumn bool
		err = d.db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('selected_movies') WHERE name='suggestion_id'").Scan(&hasColumn)
		if err != nil || !hasColumn {
			log.Println("Recriando tabela selected_movies com estrutura correta...")
			_, err = d.db.Exec("DROP TABLE IF EXISTS selected_movies")
			if err != nil {
				return err
			}
		}
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS suggestions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			movie_name TEXT NOT NULL,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			suggested_at TIMESTAMP NOT NULL,
			tmdb_id INTEGER UNIQUE,
			rating REAL,
			genres TEXT,
			release_year TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS selected_movies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			suggestion_id INTEGER NOT NULL,
			selected_at TIMESTAMP NOT NULL,
			FOREIGN KEY (suggestion_id) REFERENCES suggestions (id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS movie_reviews (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			suggestion_id INTEGER NOT NULL,
			user_id TEXT NOT NULL,
			username TEXT NOT NULL,
			rating REAL NOT NULL CHECK(rating >= 0 AND rating <= 10),
			review_text TEXT,
			reviewed_at TIMESTAMP NOT NULL,
			FOREIGN KEY (suggestion_id) REFERENCES suggestions (id) ON DELETE CASCADE,
			UNIQUE(suggestion_id, user_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_suggestions_user_id ON suggestions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_suggestions_tmdb_id ON suggestions(tmdb_id)`,
		`CREATE INDEX IF NOT EXISTS idx_selected_movies_suggestion_id ON selected_movies(suggestion_id)`,
		`CREATE INDEX IF NOT EXISTS idx_movie_reviews_suggestion_id ON movie_reviews(suggestion_id)`,
		`CREATE INDEX IF NOT EXISTS idx_movie_reviews_user_id ON movie_reviews(user_id)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			log.Printf("Erro ao executar query: %s\nErro: %v", query, err)
			return err
		}
	}

	log.Println("Database initialized successfully")
	return nil
}

func (d *Database) MovieAlreadySuggested(tmdbID int) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM suggestions WHERE tmdb_id = ?", tmdbID).Scan(&count)
	return count > 0, err
}

func (d *Database) GetMovieSuggester(tmdbID int) (string, error) {
	var username string
	err := d.db.QueryRow("SELECT username FROM suggestions WHERE tmdb_id = ?", tmdbID).Scan(&username)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return username, err
}

func (d *Database) SaveSuggestion(s *Suggestion) (int64, error) {
	result, err := d.db.Exec(`
		INSERT INTO suggestions (movie_name, user_id, username, suggested_at, tmdb_id, rating, genres, release_year)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.MovieName, s.UserID, s.Username, time.Now(), s.TMDBID, s.Rating, s.Genres, s.ReleaseYear,
	)
	if err != nil {
		log.Printf("Erro ao salvar sugestão: %v", err)
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	log.Printf("Sugestão salva com ID: %d", id)
	return id, nil
}

func (d *Database) GetUserStats(userID string) (count int, avgRating float64, err error) {
	err = d.db.QueryRow(`
		SELECT COUNT(*), COALESCE(AVG(rating), 0)
		FROM suggestions
		WHERE user_id = ?`, userID).Scan(&count, &avgRating)
	return
}

func (d *Database) GetUserSuggestions(userID string) ([]Suggestion, error) {
	log.Printf("Buscando sugestões para user_id: %s", userID)
	
	rows, err := d.db.Query(`
		SELECT s.id, s.movie_name, s.user_id, s.username, s.suggested_at, 
		       s.tmdb_id, s.rating, s.genres, s.release_year,
		       CASE WHEN sm.id IS NOT NULL THEN 1 ELSE 0 END as is_selected
		FROM suggestions s
		LEFT JOIN selected_movies sm ON s.id = sm.suggestion_id
		WHERE s.user_id = ?
		ORDER BY s.suggested_at DESC`, userID)
	
	if err != nil {
		log.Printf("Erro na query GetUserSuggestions: %v", err)
		return nil, err
	}
	defer rows.Close()

	var suggestions []Suggestion
	for rows.Next() {
		var s Suggestion
		var isSelectedInt int
		err := rows.Scan(&s.ID, &s.MovieName, &s.UserID, &s.Username, &s.SuggestedAt,
			&s.TMDBID, &s.Rating, &s.Genres, &s.ReleaseYear, &isSelectedInt)
		if err != nil {
			log.Printf("Erro ao escanear linha: %v", err)
			return nil, err
		}
		s.IsSelected = isSelectedInt == 1
		suggestions = append(suggestions, s)
		log.Printf("Sugestão encontrada: %s (ID: %d)", s.MovieName, s.ID)
	}

	log.Printf("Total de sugestões encontradas: %d", len(suggestions))
	return suggestions, nil
}

func (d *Database) GetRandomMovie() (*MovieResult, error) {
	var m MovieResult
	err := d.db.QueryRow(`
		SELECT s.id, s.movie_name, s.username, s.rating, s.genres, s.release_year, s.tmdb_id
		FROM suggestions s
		LEFT JOIN selected_movies sm ON s.id = sm.suggestion_id
		WHERE sm.id IS NULL
		ORDER BY RANDOM()
		LIMIT 1`).Scan(&m.ID, &m.MovieName, &m.Username, &m.Rating, &m.Genres, &m.ReleaseYear, &m.TMDBID)

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
		SELECT id, movie_name, username, rating, genres, release_year, tmdb_id
		FROM suggestions
		WHERE id = ?`, suggestionID).Scan(&m.ID, &m.MovieName, &m.Username, &m.Rating, &m.Genres, &m.ReleaseYear, &m.TMDBID)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (d *Database) MarkMovieSelected(suggestionID int) error {
	_, err := d.db.Exec(`
		INSERT INTO selected_movies (suggestion_id, selected_at)
		VALUES (?, ?)`, suggestionID, time.Now())
	return err
}

func (d *Database) GetAllSuggestionsCount() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM suggestions").Scan(&count)
	return count, err
}

func (d *Database) GetSelectedMoviesCount() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM selected_movies").Scan(&count)
	return count, err
}

func (d *Database) SearchUserSuggestions(movieName, userID string) (*Suggestion, error) {
	var s Suggestion
	err := d.db.QueryRow(`
		SELECT id, movie_name, tmdb_id, user_id, username
		FROM suggestions
		WHERE LOWER(movie_name) LIKE LOWER(?) AND user_id = ?
		LIMIT 1`, "%"+movieName+"%", userID).Scan(&s.ID, &s.MovieName, &s.TMDBID, &s.UserID, &s.Username)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (d *Database) SearchAnySuggestion(movieName string) (*Suggestion, error) {
	var s Suggestion
	err := d.db.QueryRow(`
		SELECT id, movie_name, tmdb_id, user_id, username
		FROM suggestions
		WHERE LOWER(movie_name) LIKE LOWER(?)
		LIMIT 1`, "%"+movieName+"%").Scan(&s.ID, &s.MovieName, &s.TMDBID, &s.UserID, &s.Username)

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


func (d *Database) IsMovieSelected(suggestionID int) (bool, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM selected_movies WHERE suggestion_id = ?", suggestionID).Scan(&count)
	return count > 0, err
}

func (d *Database) SaveMovieReview(review *MovieReview) error {
	_, err := d.db.Exec(`
		INSERT INTO movie_reviews (suggestion_id, user_id, username, rating, review_text, reviewed_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(suggestion_id, user_id) 
		DO UPDATE SET rating = excluded.rating, review_text = excluded.review_text, reviewed_at = excluded.reviewed_at`,
		review.SuggestionID, review.UserID, review.Username, review.Rating, review.ReviewText, time.Now())
	return err
}

func (d *Database) GetMovieReviews(suggestionID int) ([]MovieReview, error) {
	rows, err := d.db.Query(`
		SELECT id, suggestion_id, user_id, username, rating, COALESCE(review_text, ''), reviewed_at
		FROM movie_reviews
		WHERE suggestion_id = ?
		ORDER BY reviewed_at DESC`, suggestionID)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviews []MovieReview
	for rows.Next() {
		var r MovieReview
		err := rows.Scan(&r.ID, &r.SuggestionID, &r.UserID, &r.Username, &r.Rating, &r.ReviewText, &r.ReviewedAt)
		if err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}

	return reviews, nil
}

func (d *Database) GetUserReview(suggestionID int, userID string) (*MovieReview, error) {
	var r MovieReview
	err := d.db.QueryRow(`
		SELECT id, suggestion_id, user_id, username, rating, COALESCE(review_text, ''), reviewed_at
		FROM movie_reviews
		WHERE suggestion_id = ? AND user_id = ?`, suggestionID, userID).Scan(
		&r.ID, &r.SuggestionID, &r.UserID, &r.Username, &r.Rating, &r.ReviewText, &r.ReviewedAt)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (d *Database) GetAverageMovieRating(suggestionID int) (float64, int, error) {
	var avg sql.NullFloat64
	var count int
	err := d.db.QueryRow(`
		SELECT AVG(rating), COUNT(*)
		FROM movie_reviews
		WHERE suggestion_id = ?`, suggestionID).Scan(&avg, &count)
	
	if err != nil {
		return 0, 0, err
	}
	
	if !avg.Valid {
		return 0, 0, nil
	}
	
	return avg.Float64, count, nil
}

func (d *Database) GetAllSelectedMovies() ([]SelectedMovieWithReviews, error) {
	rows, err := d.db.Query(`
		SELECT s.id, s.movie_name, s.username, s.rating, s.genres, s.release_year, s.tmdb_id
		FROM suggestions s
		INNER JOIN selected_movies sm ON s.id = sm.suggestion_id
		ORDER BY sm.selected_at DESC`)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []SelectedMovieWithReviews
	for rows.Next() {
		var m SelectedMovieWithReviews
		err := rows.Scan(&m.ID, &m.MovieName, &m.Username, &m.Rating, &m.Genres, &m.ReleaseYear, &m.TMDBID)
		if err != nil {
			return nil, err
		}
		
		reviews, _ := d.GetMovieReviews(m.ID)
		m.Reviews = reviews
		m.ReviewCount = len(reviews)
		
		avgRating, _, _ := d.GetAverageMovieRating(m.ID)
		m.AverageScore = avgRating
		
		movies = append(movies, m)
	}

	return movies, nil
}

func (d *Database) SearchSelectedMovie(movieName string) (*MovieResult, error) {
	var m MovieResult
	err := d.db.QueryRow(`
		SELECT s.id, s.movie_name, s.username, s.rating, s.genres, s.release_year, s.tmdb_id
		FROM suggestions s
		INNER JOIN selected_movies sm ON s.id = sm.suggestion_id
		WHERE LOWER(s.movie_name) LIKE LOWER(?)
		LIMIT 1`, "%"+movieName+"%").Scan(&m.ID, &m.MovieName, &m.Username, &m.Rating, &m.Genres, &m.ReleaseYear, &m.TMDBID)

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