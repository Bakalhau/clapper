package database

import (
	"database/sql"
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
			FOREIGN KEY (suggestion_id) REFERENCES suggestions (id)
		)`,
	}

	for _, query := range queries {
		if _, err := d.db.Exec(query); err != nil {
			return err
		}
	}

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
		return 0, err
	}
	return result.LastInsertId()
}

func (d *Database) GetUserStats(userID string) (count int, avgRating float64, err error) {
	err = d.db.QueryRow(`
		SELECT COUNT(*), COALESCE(AVG(rating), 0)
		FROM suggestions
		WHERE user_id = ?`, userID).Scan(&count, &avgRating)
	return
}

func (d *Database) GetUserSuggestions(userID string) ([]Suggestion, error) {
	rows, err := d.db.Query(`
		SELECT s.id, s.movie_name, s.user_id, s.username, s.suggested_at, 
		       s.tmdb_id, s.rating, s.genres, s.release_year,
		       CASE WHEN sm.id IS NOT NULL THEN 1 ELSE 0 END as is_selected
		FROM suggestions s
		LEFT JOIN selected_movies sm ON s.id = sm.suggestion_id
		WHERE s.user_id = ?
		ORDER BY s.suggested_at DESC`, userID)
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []Suggestion
	for rows.Next() {
		var s Suggestion
		err := rows.Scan(&s.ID, &s.MovieName, &s.UserID, &s.Username, &s.SuggestedAt,
			&s.TMDBID, &s.Rating, &s.Genres, &s.ReleaseYear, &s.IsSelected)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, s)
	}

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
		SELECT id, movie_name, tmdb_id, user_id
		FROM suggestions
		WHERE LOWER(movie_name) LIKE LOWER(?) AND user_id = ?
		LIMIT 1`, "%"+movieName+"%", userID).Scan(&s.ID, &s.MovieName, &s.TMDBID, &s.UserID)

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

	if _, err := tx.Exec("DELETE FROM selected_movies WHERE suggestion_id = ?", suggestionID); err != nil {
		return err
	}
	if _, err := tx.Exec("DELETE FROM suggestions WHERE id = ?", suggestionID); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) Close() error {
	return d.db.Close()
}