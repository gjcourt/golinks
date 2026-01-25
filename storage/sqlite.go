package storage

import (
	"database/sql"
	"errors"
	"time"

	"github.com/george/golinks/models"
	_ "github.com/mattn/go-sqlite3"
)

// ErrNotFound is returned when a link is not found
var ErrNotFound = errors.New("link not found")

// ErrAlreadyExists is returned when a link already exists
var ErrAlreadyExists = errors.New("link already exists")

// SQLiteStore implements Store using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-based store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	schema := `
	CREATE TABLE IF NOT EXISTS links (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		shortcode TEXT UNIQUE NOT NULL,
		url TEXT NOT NULL,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		click_count INTEGER DEFAULT 0,
		last_clicked DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_shortcode ON links(shortcode);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// CreateLink creates a new link
func (s *SQLiteStore) CreateLink(link *models.Link) error {
	now := time.Now()
	link.CreatedAt = now
	link.UpdatedAt = now

	result, err := s.db.Exec(
		"INSERT INTO links (shortcode, url, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		link.Shortcode, link.URL, link.Description, link.CreatedAt, link.UpdatedAt,
	)
	if err != nil {
		// Check if it's a unique constraint violation
		if err.Error() == "UNIQUE constraint failed: links.shortcode" {
			return ErrAlreadyExists
		}
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	link.ID = id

	return nil
}

// GetLink retrieves a link by shortcode
func (s *SQLiteStore) GetLink(shortcode string) (*models.Link, error) {
	link := &models.Link{}
	err := s.db.QueryRow(
		"SELECT id, shortcode, url, description, created_at, updated_at, click_count FROM links WHERE shortcode = ?",
		shortcode,
	).Scan(&link.ID, &link.Shortcode, &link.URL, &link.Description, &link.CreatedAt, &link.UpdatedAt, &link.ClickCount)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return link, nil
}

// UpdateLink updates an existing link
func (s *SQLiteStore) UpdateLink(shortcode string, link *models.Link) error {
	link.UpdatedAt = time.Now()

	result, err := s.db.Exec(
		"UPDATE links SET url = ?, description = ?, updated_at = ? WHERE shortcode = ?",
		link.URL, link.Description, link.UpdatedAt, shortcode,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteLink deletes a link by shortcode
func (s *SQLiteStore) DeleteLink(shortcode string) error {
	result, err := s.db.Exec("DELETE FROM links WHERE shortcode = ?", shortcode)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

// ListLinks returns all links
func (s *SQLiteStore) ListLinks() ([]*models.Link, error) {
	rows, err := s.db.Query(
		"SELECT id, shortcode, url, description, created_at, updated_at, click_count FROM links ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var links []*models.Link
	for rows.Next() {
		link := &models.Link{}
		err := rows.Scan(&link.ID, &link.Shortcode, &link.URL, &link.Description, &link.CreatedAt, &link.UpdatedAt, &link.ClickCount)
		if err != nil {
			return nil, err
		}
		links = append(links, link)
	}

	return links, nil
}

// IncrementClickCount increments the click count for a link
func (s *SQLiteStore) IncrementClickCount(shortcode string) error {
	_, err := s.db.Exec(
		"UPDATE links SET click_count = click_count + 1, last_clicked = ? WHERE shortcode = ?",
		time.Now(), shortcode,
	)
	return err
}

// GetStats returns statistics for a link
func (s *SQLiteStore) GetStats(shortcode string) (*models.LinkStats, error) {
	stats := &models.LinkStats{Shortcode: shortcode}
	var lastClicked sql.NullTime

	err := s.db.QueryRow(
		"SELECT click_count, last_clicked FROM links WHERE shortcode = ?",
		shortcode,
	).Scan(&stats.ClickCount, &lastClicked)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if lastClicked.Valid {
		stats.LastClicked = lastClicked.Time
	}

	return stats, nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
