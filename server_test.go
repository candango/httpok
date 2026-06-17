package httpok

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func isPortInUse(port int) (bool, error) {
	// Try to listen on the specified port
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))

	if err != nil {
		// If there's an error, check if it's because the port is already in use
		if opErr, ok := err.(*net.OpError); ok && opErr.Op == "listen" {
			return true, nil
		}
		// Return the error if it's something else
		return false, err
	}
	// If we can listen, the port is not in use, so close the listener
	ln.Close()
	return false, nil
}

// Helper function to get a free port
func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func waitForPortInUse(t *testing.T, port int, want bool) {
	t.Helper()

	deadline := time.After(2 * time.Second)
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-deadline:
			ok, err := isPortInUse(port)
			if err != nil {
				t.Fatalf("failed to check port in use: %v", err)
			}
			if ok != want {
				t.Fatalf("expected port in use to be %v, got it %v", want, ok)
			}
			return
		case <-tick.C:
			ok, err := isPortInUse(port)
			if err != nil {
				t.Fatalf("failed to check port in use: %v", err)
			}
			if ok == want {
				return
			}
		}
	}
}

func TestGracefulServerSignalCancelsRuntimeContext(t *testing.T) {
	// Generate a random free port
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	addr := fmt.Sprintf(":%d", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Request received")
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	cancelFuncCalled := make(chan struct{})

	// Create GracefulServer with our custom server
	gs := NewGracefulServer(srv, "test-server").WithCancelFunc(
		func(ctx context.Context) error {
			close(cancelFuncCalled)
			return nil
		},
	)

	done := make(chan struct{})
	go func() {
		gs.Run(syscall.SIGUSR1)
		close(done)
	}()

	waitForPortInUse(t, port, true)

	err = syscall.Kill(os.Getpid(), syscall.SIGUSR1)
	assert.NoError(t, err)

	select {
	case <-cancelFuncCalled:
	case <-time.After(1 * time.Second):
		t.Fatal("expected cancel function to be called")
	}

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("expected server shutdown to finish")
	}

	select {
	case <-gs.Done():
	case <-time.After(1 * time.Second):
		t.Fatal("expected runtime context to be canceled")
	}
}

func TestGracefulServer(t *testing.T) {
	// Generate a random free port
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get free port: %v", err)
	}
	addr := fmt.Sprintf(":%d", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Request received")
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Create GracefulServer with our custom server
	gs := NewGracefulServer(srv, "test-server")
	go func() {
		gs.Run()
	}()

	waitForPortInUse(t, port, true)

	resp, err := http.Get("http://localhost" + addr)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Triggering shutdown
	gs.TriggerShutdown()
	waitForPortInUse(t, port, false)

	resp, err = http.Get("http://localhost" + addr)
	if err != nil {
		assert.Error(t, err)
	}

	waitForPortInUse(t, port, false)
}
