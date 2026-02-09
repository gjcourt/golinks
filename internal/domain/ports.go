package domain

// LinkRepository is the driven port for persisting and retrieving links.
// Implementations live in adapter/postgres, adapter/sqlite, etc.
type LinkRepository interface {
	CreateLink(link *Link) error
	GetLink(shortcode string) (*Link, error)
	UpdateLink(shortcode string, link *Link) error
	DeleteLink(shortcode string) error
	ListLinks() ([]*Link, error)
	IncrementClickCount(shortcode string) error
	GetStats(shortcode string) (*LinkStats, error)
	Close() error
}

// LinkService is the driving port consumed by HTTP handlers and other
// entry-points. It encapsulates all business rules.
type LinkService interface {
	CreateLink(shortcode, url, description string) (*Link, error)
	GetLink(shortcode string) (*Link, error)
	UpdateLink(shortcode, url, description string) (*Link, error)
	DeleteLink(shortcode string) error
	ListLinks() ([]*Link, error)
	RedirectLink(shortcode string) (*Link, error)
	GetLinkStats(shortcode string) (*LinkStats, error)
}
