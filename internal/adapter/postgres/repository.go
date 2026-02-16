// Package postgres implements the link repository using PostgreSQL.
package postgres

import (
	"database/sql"
	"time"

	"github.com/george/golinks/internal/domain"
	"github.com/lib/pq"
)

// Repository implements domain.LinkRepository using PostgreSQL.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new PostgreSQL-based repository.
func NewRepository(connStr string) (*Repository, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	schema := `
	CREATE TABLE IF NOT EXISTS links (
		id BIGSERIAL PRIMARY KEY,
		shortcode TEXT UNIQUE NOT NULL,
		url TEXT NOT NULL,
		description TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW(),
		click_count INTEGER DEFAULT 0,
		last_clicked TIMESTAMPTZ
	);
	CREATE INDEX IF NOT EXISTS idx_shortcode ON links(shortcode);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

// CreateLink adds a new link to the database.
func (r *Repository) CreateLink(link *domain.Link) error {
	now := time.Now()
	link.CreatedAt = now
	link.UpdatedAt = now
	err := r.db.QueryRow(
		"INSERT INTO links (shortcode, url, description, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		link.Shortcode, link.URL, link.Description, link.CreatedAt, link.UpdatedAt,
	).Scan(&link.ID)
	if err != nil {
		if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return err
	}
	return nil
}

// GetLink retrieves a link by its shortcode.
func (r *Repository) GetLink(shortcode string) (*domain.Link, error) {
	link := &domain.Link{}
	err := r.db.QueryRow(
		"SELECT id, shortcode, url, description, created_at, updated_at, click_count FROM links WHERE shortcode = $1",
		shortcode,
	).Scan(&link.ID, &link.Shortcode, &link.URL, &link.Description, &link.CreatedAt, &link.UpdatedAt, &link.ClickCount)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return link, err
}

// UpdateLink updates an existing link.
func (r *Repository) UpdateLink(shortcode string, link *domain.Link) error {
	link.UpdatedAt = time.Now()
	result, err := r.db.Exec(
		"UPDATE links SET url = $1, description = $2, updated_at = $3 WHERE shortcode = $4",
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
		return domain.ErrNotFound
	}
	return nil
}

// DeleteLink deletes a link.
func (r *Repository) DeleteLink(shortcode string) error {
	result, err := r.db.Exec("DELETE FROM links WHERE shortcode = $1", shortcode)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ListLinks lists all links.
func (r *Repository) ListLinks() ([]*domain.Link, error) {
	rows, err := r.db.Query(
		"SELECT id, shortcode, url, description, created_at, updated_at, click_count FROM links ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var links []*domain.Link
	for rows.Next() {
		link := &domain.Link{}
		if err := rows.Scan(&link.ID, &link.Shortcode, &link.URL, &link.Description, &link.CreatedAt, &link.UpdatedAt, &link.ClickCount); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

// IncrementClickCount increments the click count for a link.
func (r *Repository) IncrementClickCount(shortcode string) error {
	_, err := r.db.Exec(
		"UPDATE links SET click_count = click_count + 1, last_clicked = $1 WHERE shortcode = $2",
		time.Now(), shortcode,
	)
	return err
}

// GetStats retrieves statistics for a link.
func (r *Repository) GetStats(shortcode string) (*domain.LinkStats, error) {
	stats := &domain.LinkStats{Shortcode: shortcode}
	var lastClicked sql.NullTime
	err := r.db.QueryRow(
		"SELECT click_count, last_clicked FROM links WHERE shortcode = $1",
		shortcode,
	).Scan(&stats.ClickCount, &lastClicked)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if lastClicked.Valid {
		stats.LastClicked = lastClicked.Time
	}
	return stats, nil
}

// Close closes the database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}
