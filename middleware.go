// Copyright 2023-2024 Flavio Garcia
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
	"net/http"
)

// Middleware represents a function that can wrap a http.Handler with
// additional functionality. It takes a http.Handler and returns a new
// http.Handler that includes the middleware's behavior.
type Middleware func(http.Handler) http.Handler

// Chain creates a chain of HTTP middleware functions to wrap around a
// http.Handler.
// It applies each middleware in the order they are provided, allowing for
// layered processing of HTTP requests and responses.
func Chain(next http.Handler, ms ...Middleware) http.Handler {
	for i := len(ms) - 1; i >= 0; i-- {
		m := ms[i]
		next = m(next)
	}
	return next
}
