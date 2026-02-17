// Package sqlite implements the link repository using SQLite.
package sqlite

import (
	"database/sql"
	"time"

	"github.com/george/golinks/internal/domain"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// Repository implements domain.LinkRepository using SQLite.
type Repository struct {
	db *sql.DB
}

// UserRepository implements domain.UserRepository using SQLite.
type UserRepository struct {
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
		owner TEXT NOT NULL DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		click_count INTEGER DEFAULT 0,
		last_clicked DATETIME
	);
	CREATE INDEX IF NOT EXISTS idx_shortcode ON links(shortcode);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		role TEXT NOT NULL DEFAULT 'user',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return &Repository{db: db}, nil
}

// NewUserRepository creates a UserRepository that shares the same DB.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// DB returns the underlying database connection.
func (r *Repository) DB() *sql.DB {
	return r.db
}

// CreateLink adds a new link to the database.
func (r *Repository) CreateLink(link *domain.Link) error {
	now := time.Now()
	link.CreatedAt = now
	link.UpdatedAt = now
	result, err := r.db.Exec(
		"INSERT INTO links (shortcode, url, description, owner, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		link.Shortcode, link.URL, link.Description, link.Owner, link.CreatedAt, link.UpdatedAt,
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

// GetLink retrieves a link by its shortcode.
func (r *Repository) GetLink(shortcode string) (*domain.Link, error) {
	link := &domain.Link{}
	err := r.db.QueryRow(
		"SELECT id, shortcode, url, description, owner, created_at, updated_at, click_count FROM links WHERE shortcode = ?",
		shortcode,
	).Scan(&link.ID, &link.Shortcode, &link.URL, &link.Description, &link.Owner, &link.CreatedAt, &link.UpdatedAt, &link.ClickCount)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return link, err
}

// UpdateLink updates an existing link.
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

// DeleteLink deletes a link.
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

// ListLinks lists all links.
func (r *Repository) ListLinks() ([]*domain.Link, error) {
	rows, err := r.db.Query(
		"SELECT id, shortcode, url, description, owner, created_at, updated_at, click_count FROM links ORDER BY created_at DESC",
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var links []*domain.Link
	for rows.Next() {
		link := &domain.Link{}
		if err := rows.Scan(&link.ID, &link.Shortcode, &link.URL, &link.Description, &link.Owner, &link.CreatedAt, &link.UpdatedAt, &link.ClickCount); err != nil {
			return nil, err
		}
		links = append(links, link)
	}
	return links, nil
}

// IncrementClickCount increments the click count for a link.
func (r *Repository) IncrementClickCount(shortcode string) error {
	_, err := r.db.Exec(
		"UPDATE links SET click_count = click_count + 1, last_clicked = ? WHERE shortcode = ?",
		time.Now(), shortcode,
	)
	return err
}

// GetStats retrieves statistics for a link.
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

// Close closes the database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}

// ---------------------------------------------------------------------------
// UserRepository
// ---------------------------------------------------------------------------

// CreateUser inserts a new user.
func (ur *UserRepository) CreateUser(user *domain.User) error {
	user.CreatedAt = time.Now()
	result, err := ur.db.Exec(
		"INSERT INTO users (username, password_hash, role, created_at) VALUES (?, ?, ?, ?)",
		user.Username, user.PasswordHash, user.Role, user.CreatedAt,
	)
	if err != nil {
		if err.Error() == "UNIQUE constraint failed: users.username" {
			return domain.ErrAlreadyExists
		}
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id
	return nil
}

// GetUserByUsername retrieves a user by username.
func (ur *UserRepository) GetUserByUsername(username string) (*domain.User, error) {
	u := &domain.User{}
	err := ur.db.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	return u, err
}

// CountUsers returns the total number of users.
func (ur *UserRepository) CountUsers() (int64, error) {
	var n int64
	err := ur.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&n)
	return n, err
}

// Close is a no-op; the underlying DB is owned by Repository.
func (ur *UserRepository) Close() error { return nil }

// HashPassword hashes a plaintext password with bcrypt.
func HashPassword(password string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(h), err
}

// CheckPassword compares a plaintext password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
