package config

import "time"

// Config captures server-wide configuration knobs. Values are intentionally
// minimal for now; future phases will extend this struct in-place.
type Config struct {
	GRPCPort          int
	MaxSessions       int
	IdleSessionExpiry time.Duration
	DataDir           string
	BufferPoolPages   int
}

// Load returns default configuration. Later this will consult files/env/etcd.
func Load() Config {
	return Config{
		GRPCPort:          7000,
		MaxSessions:       128,
		IdleSessionExpiry: 5 * time.Minute,
		DataDir:           "data",
		BufferPoolPages:   64,
	}
}
