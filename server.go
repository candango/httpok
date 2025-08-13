package httpok

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/candango/httpok/logger"
	"github.com/candango/httpok/session"
)

// newSignalChan creates a channel that listens for specified signals or
// default signals if none are provided.
// It returns a channel that receives these signals. This function is used
// internally by [GracefulServer.Run]
func newSignalChan(sig ...os.Signal) chan os.Signal {
	if len(sig) == 0 {
		sig = []os.Signal{
			syscall.SIGINT,
			syscall.SIGHUP,
			syscall.SIGQUIT,
			syscall.SIGTERM,
		}
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig...)
	return c
}

// GracefulServer combines an HTTP server with a context for graceful shutdown
// handling.
type GracefulServer struct {
	Name string
	*http.Server
	context.Context
	logger.Logger
	SessionEngine session.Engine
	sigChan       chan os.Signal
}

func NewGracefulServer(s *http.Server) *GracefulServer {
	gs := &GracefulServer{
		Server:  s,
		Context: context.Background(),
	}
	return gs
}

// Run starts the HTTP server in a goroutine and listens for termination
// signals to gracefully shut down.
// It takes optional signals to listen for; if none are provided, it uses
// default signals.
func (s *GracefulServer) Run(sig ...os.Signal) {
	l := s.Logger
	if l == nil {
		l = &logger.StandardLogger{}
	}

	go func() {
		err := s.ListenAndServe()
		if err != http.ErrServerClosed {
			l.Fatalf("server %s HTTP ListenAndServe error: %v", s.Name, err)
		}
	}()
	ctx, cancel := context.WithCancel(s.Context)

	l.Printf("server %s started at %s", s.Name, s.Addr)
	c := newSignalChan(sig...)
	go func() {
		defer func() {
			signal.Stop(c)
			cancel()
		}()

		select {
		case sig := <-c:
			l.Printf("shutting down %s due to signal %s", s.Name, sig)
		case <-ctx.Done():
			l.Printf("shutting down %s", s.Name)
		}

		if err := s.Shutdown(ctx); err != nil {
			l.Fatalf("Server %s shutdown failed: %v", s.Name, err)
		}

		l.Printf("%s shutdown gracefully", s.Name)
	}()
	<-done
}
