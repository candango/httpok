package httpok

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
			syscall.SIGUSR1,
			syscall.SIGUSR2,
		}
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig...)
	return c
}

// GracefulCancelFunc defines a user-provided function called during graceful
// shutdown for custom cleanup. The provided context is scoped to the shutdown
// phase and may include the configured shutdown timeout.
type GracefulCancelFunc func(context.Context) error

// GracefulServer combines an HTTP server with a runtime context for graceful
// shutdown handling. The embedded context is canceled when shutdown is
// triggered by a signal or by TriggerShutdown.
type GracefulServer struct {
	Name string
	*http.Server
	context.Context
	logger.Logger
	SessionEngine   session.Engine
	ShutdownTimeout float64
	cancel          context.CancelFunc
	CancelFunc      GracefulCancelFunc
	cancelMutex     sync.Mutex
	sigChan         chan os.Signal
}

// NewGracefulServer creates a new GracefulServer wrapping the given http.Server.
// It initializes a cancelable runtime context used to signal shutdown to
// dependents.
func NewGracefulServer(s *http.Server, name string) *GracefulServer {
	ctx, cancel := context.WithCancel(context.Background())
	gs := &GracefulServer{
		Name:    name,
		Server:  s,
		Context: ctx,
		cancel:  cancel,
	}
	return gs
}

// WithShutdownTimeout sets the shutdown timeout (in seconds) for the server.
// Returns the server for method chaining.
func (s *GracefulServer) WithShutdownTimeout(timeout float64) *GracefulServer {
	s.ShutdownTimeout = timeout
	return s
}

// WithCancelFunc sets a custom cleanup function called during graceful
// shutdown before the HTTP server is shut down. The function receives the
// shutdown context, not the runtime context.
func (s *GracefulServer) WithCancelFunc(cancelFunc GracefulCancelFunc) *GracefulServer {
	s.CancelFunc = cancelFunc
	return s
}

// TriggerShutdown programmatically requests a graceful shutdown of the server.
// Returns an error if the shutdown cancel function is not available (e.g., Run
// has not been called).
func (s *GracefulServer) TriggerShutdown() error {
	s.cancelMutex.Lock()
	defer s.cancelMutex.Unlock()
	if s.cancel == nil {
		return fmt.Errorf("no shutdown cancel function available")
	}
	s.cancel()
	return nil
}

// Run starts the HTTP server in a goroutine and listens for termination
// signals to gracefully shut down.
// It cancels the server runtime context when shutdown is triggered, then runs
// the custom cancel function and HTTP shutdown using a separate shutdown
// context. If ShutdownTimeout is set, that timeout applies to the shutdown
// context.
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

	l.Printf("server %s started at %s", s.Name, s.Addr)
	s.sigChan = newSignalChan(sig...)
	done := make(chan struct{})
	s.cancelMutex.Lock()
	if s.Context == nil {
		s.Context = context.Background()
	}
	if s.cancel == nil {
		s.Context, s.cancel = context.WithCancel(s.Context)
	}
	ctx := s.Context
	cancel := s.cancel
	s.cancelMutex.Unlock()

	go func() {
		select {
		case sig := <-s.sigChan:
			l.Printf("shutting down %s due to signal %s", s.Name, sig)
			cancel()
		case <-ctx.Done():
			l.Printf("shutting down %s cancellation triggered", s.Name)
		}

		shutdownCtx := context.Background()
		shutdownCancel := func() {}
		if s.ShutdownTimeout > 0 {
			shutdownCtx, shutdownCancel = context.WithTimeout(shutdownCtx,
				time.Duration(s.ShutdownTimeout)*time.Second)
		}
		defer shutdownCancel()

		defer func() {
			signal.Stop(s.sigChan)
			cancel()
			close(done)
		}()

		if s.CancelFunc != nil {
			if err := s.CancelFunc(shutdownCtx); err != nil {
				l.Fatalf("Server %s cancellation function failed: %v", s.Name, err)
			}
		}

		if err := s.Server.Shutdown(shutdownCtx); err != nil {
			l.Fatalf("Server %s shutdown failed: %v", s.Name, err)
		}

		l.Printf("%s shutdown gracefully", s.Name)
	}()
	<-done
}
