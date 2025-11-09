package server

import (
	"context"
	"log"

	"github.com/hainn191297/myDb/internal/config"
	"github.com/hainn191297/myDb/internal/server/session"
)

// Server wires together transport (gRPC) and session handling.
type Server struct {
	cfg     config.Config
	sessMgr *session.Manager
}

// New constructs a server with default middleware wired.
func New(cfg config.Config) *Server {
	return &Server{
		cfg:     cfg,
		sessMgr: session.NewManager(cfg.MaxSessions, cfg.IdleSessionExpiry),
	}
}

// Start simulates serving traffic until the context is canceled. Real gRPC
// wiring will replace this stub once dependencies are introduced.
func (s *Server) Start(ctx context.Context) error {
	log.Printf("myDb: stub server listening on :%d", s.cfg.GRPCPort)
	<-ctx.Done()

	if err := ctx.Err(); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		return err
	}
	return nil
}

// itoa avoids importing strconv everywhere until dependencies settle.
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	buf := make([]byte, 0, 12)
	for v > 0 {
		buf = append([]byte{byte('0' + v%10)}, buf...)
		v /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
