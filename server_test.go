// Copyright 2023-2025 Flavio Garcia
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpok

import (
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
	gs := NewGracefulServer(srv)
	var c chan os.Signal
	go func() {
		c = gs.Run()
	}()

	time.Sleep(1 * time.Second)
	ok, err := isPortInUse(port)
	if err != nil {
		t.Fatalf("failed to check port in use: %v", err)
	}
	assert.True(t, ok)

	resp, err := http.Get("http://localhost" + addr)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	// Send SIGTERM down the pipe
	c <- syscall.SIGTERM
	time.Sleep(1 * time.Second)

	resp, err = http.Get("http://localhost" + addr)
	if err != nil {
		assert.Error(t, err)
	}

	ok, err = isPortInUse(port)
	if err != nil {
		t.Fatalf("failed to check port in use: %v", err)
	}
	assert.False(t, ok)
}
