package spec

// BindTo binds to either a Unix socket or a TCP interface
type BindTo struct {
	Network string // "unix" or "tcp"
	Address string // unix file path, or <addr>:<port>
}
