package sqlite

import (
	"database/sql"
	"time"

	"github.com/george/golinks/internal/domain"
	_ "github.com/mattn/go-sqlite3"
)

// Repository implements domain.LinkRepository using SQLite.
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new SQLite-based repository.
func NewRepository(dbPath string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
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
	return &Repository{db: db}, nil
}

func (r *Repository) CreateLink(link *domain.Link) error {
	now := time.Now()
	link.CreatedAt = now
	link.UpdatedAt = now
	result, err := r.db.Exec(
		"INSERT INTO links (shortcode, url, description, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		link.Shortcode, link.URL, link.Description, link.CreatedAt, link.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: links.shortcode" {
			return domain.ErrAlreadyExists
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

func (r *Repository) GetLink(shortcode string) (*domain.Link, error) {
	link := &domain.Link{}
	err := r.db.QueryRow(
		"SELECT id, shortcode, url, description, created_at, updated_at, click_count FROM links WHERE shortcode = ?",
		shortcode,
	).Scan(&link.ID, &link.Shortcode, &link.URL, &link.Description, &link.CreatedAt, &link.UpdatedAt, &link.ClickCount)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return link, err
}

func (r *Repository) UpdateLink(shortcode string, link *domain.Link) error {
	link.UpdatedAt = time.Now()
	result, err := r.db.Exec(
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
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteLink(shortcode string) error {
	result, err := r.db.Exec("DELETE FROM links WHERE shortcode = ?", shortcode)
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

func (r *Repository) ListLinks() ([]*domain.Link, error) {
	rows, err := r.db.Query(
		"SELECT id, shortcode, url, description, created_at, updated_at, click_count FROM links ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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

func (r *Repository) IncrementClickCount(shortcode string) error {
	_, err := r.db.Exec(
		"UPDATE links SET click_count = click_count + 1, last_clicked = ? WHERE shortcode = ?",
		time.Now(), shortcode,
	)
	return err
}

func (r *Repository) GetStats(shortcode string) (*domain.LinkStats, error) {
	stats := &domain.LinkStats{Shortcode: shortcode}
	var lastClicked sql.NullTime
	err := r.db.QueryRow(
		"SELECT click_count, last_clicked FROM links WHERE shortcode = ?",
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

func (r *Repository) Close() error {
	return r.db.Close()
}
