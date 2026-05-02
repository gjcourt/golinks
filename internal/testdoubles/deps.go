package testdoubles

// ServerDeps aggregates all outbound-port fakes for unit tests.
// Add one field per outbound port as ports are extracted from domain/ into a
// formal internal/ports/{inbound,outbound}/ layout.
type ServerDeps struct{}

// NewServerDeps returns a ServerDeps with all fakes initialised to safe zero-value defaults.
func NewServerDeps() *ServerDeps {
	return &ServerDeps{}
}
