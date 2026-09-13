/*
 * Copyright (c) 2025 Sharon Aicler (saichler@gmail.com)
 *
 * Layer 8 Ecosystem is licensed under the Apache License, Version 2.0.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// CoockieToken.go provides token extraction utilities for HTTP requests.
// It supports multiple methods of providing authentication tokens:
// 1. HTTP-only cookies (primary method for browser security)
// 2. Authorization header with Bearer scheme (for API clients)
// 3. Query parameter fallback (for initial page load redirects)

package server

import (
	"net/http"
	"strings"
)

// BearerCookieName is the name of the HTTP-only cookie used to store
// bearer tokens for browser-based authentication.
var BearerCookieName = "bToken"

// extractToken attempts to extract an authentication token from an HTTP request.
// It checks multiple sources in priority order:
// 1. Cookie named "bToken" (primary method for browser security with HttpOnly flag)
// 2. Authorization header with "Bearer" scheme (for API clients)
// 3. Query parameter named "token" (fallback for redirects)
//
// Returns an empty string if no token is found in any location.
func extractToken(r *http.Request) string {
	// 1. Try cookie first (primary method for browser requests)
	cookie, err := r.Cookie(BearerCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// 2. Fallback to Authorization header (for API clients)
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// 3. Fallback to query parameter (for initial page load redirect)
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	return ""
}
