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

// LoadWebUI.go provides web UI file serving functionality for the REST server.
// It dynamically scans a "web" directory and registers HTTP handlers for all
// files found, with special handling for:
//   - index.html files at directory roots (registered as directory paths)
//   - HTML files (registered with cache-busting headers)
//   - Static assets (CSS, JS, images, etc.)
//
// The smart root handler provides SPA (Single Page Application) support by
// serving index.html for unmatched routes, while still correctly routing
// API endpoints based on the configured prefix.

package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	// webUIFileMap maps URL paths to filesystem paths for web UI files.
	webUIFileMap = make(map[string]string)
	// webUIFileMapMutex protects concurrent access to webUIFileMap.
	webUIFileMapMutex sync.RWMutex
	// webUIHandlerRegistry tracks registered HTTP handlers to prevent duplicates.
	webUIHandlerRegistry = make(map[string]http.HandlerFunc)
	// webUIHandlerRegistryMutex protects concurrent access to webUIHandlerRegistry.
	webUIHandlerRegistryMutex sync.RWMutex
	// rootHandlerRegistered tracks whether the root "/" handler has been registered.
	rootHandlerRegistered = false
)

// LoadWebUI scans the web directory and registers HTTP handlers for all files.
// It clears the file map (for hot-reload) but preserves handler registrations
// since Go's ServeMux doesn't support handler removal. In proxy mode, the root
// handler is not registered to avoid conflicts with the reverse proxy.
func (this *RestServer) LoadWebUI() {
	fmt.Println("Loading UI...")

	// Clear and reload web UI file mappings (but keep handler registry intact)
	webUIFileMapMutex.Lock()
	webUIFileMap = make(map[string]string)
	webUIFileMapMutex.Unlock()

	// DO NOT clear handler registry - handlers remain registered in ServeMux

	// Determine the web directory path
	webDir := this.getWebDirectory()

	// Scan and register all web files (non-root index.html files get handlers here)
	this.loadWebDir("/", webDir)

	// Register all .html files (except root index.html) before the root handler
	this.registerHTMLHandlers()

	// Register smart root handler LAST (only once) so specific paths are matched first
	// Skip in proxy mode - the proxy handles the root path
	if !rootHandlerRegistered && !proxyMode {
		http.HandleFunc("/", this.smartRootHandler)
		rootHandlerRegistered = true
	}
}

// getWebDirectory searches for the web directory in common locations.
// It checks: "web", "./web", "../web", "../../web" and returns the first
// found path. Defaults to "web" if none are found.
func (this *RestServer) getWebDirectory() string {
	// Try to find web directory in various locations
	possiblePaths := []string{
		"web",           // Current directory
		"./web",         // Relative to current
		"../web",        // Up one level
		"../../web",     // Up two levels
	}
	
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// Default to "web" if not found
	return "web"
}

// loadWebDir recursively scans a directory and registers file handlers.
// For index.html files, it registers the directory path as the URL.
// For other files, it registers the full file path. Non-HTML files get
// handlers immediately; HTML files are registered later in registerHTMLHandlers.
func (this *RestServer) loadWebDir(path string, webDir string) {
	dirName := concat(webDir, path)
	files, err := os.ReadDir(dirName)
	if err != nil {
		fmt.Println("Error loading web UI:", err)
		return
	}

	for _, file := range files {
		webPath := concat(path, file.Name())
		if file.IsDir() {
			this.loadWebDir(concat(webPath, "/"), webDir)
		} else {
			fullFilePath := filepath.Join(webDir, path, file.Name())
			if file.Name() == "index.html" {
				indexPath := path
				if indexPath != "/" && !strings.HasSuffix(indexPath, "/") {
					indexPath += "/"
				}
				// In proxy mode, register root index.html as "/index.html" instead of "/"
				if proxyMode && indexPath == "/" {
					indexPath = "/index.html"
				}
				fmt.Println("Loaded index.html at path:", indexPath)
				// Store mapping
				webUIFileMapMutex.Lock()
				webUIFileMap[indexPath] = fullFilePath
				webUIFileMapMutex.Unlock()

				// Don't register handlers for index.html files - let smartRootHandler handle them
				// Only register specific handlers for non-root index.html files (or proxy mode root)
				if indexPath != "/" {
					webUIHandlerRegistryMutex.RLock()
					_, exists := webUIHandlerRegistry[indexPath]
					webUIHandlerRegistryMutex.RUnlock()

					if !exists {
						handler := this.createDynamicHandler(indexPath)
						webUIHandlerRegistryMutex.Lock()
						webUIHandlerRegistry[indexPath] = handler
						webUIHandlerRegistryMutex.Unlock()
						http.HandleFunc(indexPath, handler)
					}
				}
			} else {
				fmt.Println("Loaded file:", webPath)
				// Store mapping
				webUIFileMapMutex.Lock()
				webUIFileMap[webPath] = fullFilePath
				webUIFileMapMutex.Unlock()

				// Register handlers for all non-HTML files immediately
				// HTML files (except index.html) will be registered in registerHTMLHandlers
				if !strings.HasSuffix(webPath, ".html") {
					webUIHandlerRegistryMutex.RLock()
					_, exists := webUIHandlerRegistry[webPath]
					webUIHandlerRegistryMutex.RUnlock()

					if !exists {
						handler := this.createDynamicHandler(webPath)
						webUIHandlerRegistryMutex.Lock()
						webUIHandlerRegistry[webPath] = handler
						webUIHandlerRegistryMutex.Unlock()
						http.HandleFunc(webPath, handler)
					}
				}
			}
		}
	}
}

// registerHTMLHandlers registers HTTP handlers for all .html files (except
// index.html files which are handled by loadWebDir). This is called after
// loadWebDir to ensure HTML handlers are registered before the root handler.
func (this *RestServer) registerHTMLHandlers() {
	webUIFileMapMutex.RLock()
	defer webUIFileMapMutex.RUnlock()

	for webPath := range webUIFileMap {
		// Only register handlers for .html files (excluding index.html paths)
		if strings.HasSuffix(webPath, ".html") && !strings.HasSuffix(webPath, "/") {
			webUIHandlerRegistryMutex.RLock()
			_, exists := webUIHandlerRegistry[webPath]
			webUIHandlerRegistryMutex.RUnlock()

			if !exists {
				handler := this.createDynamicHandler(webPath)
				webUIHandlerRegistryMutex.Lock()
				webUIHandlerRegistry[webPath] = handler
				webUIHandlerRegistryMutex.Unlock()
				http.HandleFunc(webPath, handler)
				fmt.Println("Registered HTML handler:", webPath)
			}
		}
	}
}

// createDynamicHandler creates an HTTP handler function for a specific path.
// The handler looks up the current file path at runtime (supporting hot-reload)
// and serves the file with cache-busting headers to ensure fresh content.
func (this *RestServer) createDynamicHandler(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Dynamically look up the current file path
		webUIFileMapMutex.RLock()
		filePath, exists := webUIFileMap[path]
		webUIFileMapMutex.RUnlock()
		
		if exists {
			// Add cache-busting headers
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, filePath)
		} else {
			// Custom 404 response
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("File Not Found"))
		}
	}
}

// smartRootHandler is the catch-all handler for the root path and unmatched routes.
// It provides SPA (Single Page Application) support by:
// 1. Passing through API endpoints (those with the configured prefix) to return 404
// 2. Serving exact file matches from the web UI map
// 3. Serving index.html for the root path
// 4. Returning 404 for all other unmatched paths
func (this *RestServer) smartRootHandler(w http.ResponseWriter, r *http.Request) {
	// Check if this looks like an API endpoint (has prefix)
	if this.Prefix != "" && strings.HasPrefix(r.URL.Path, this.Prefix) {
		// This is likely an API endpoint, let it pass through (404 will be handled by API)
		http.NotFound(w, r)
		return
	}
	
	webUIFileMapMutex.RLock()
	
	// Check for exact file match first
	filePath, exists := webUIFileMap[r.URL.Path]
	if exists {
		webUIFileMapMutex.RUnlock()
		// Add cache-busting headers
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		http.ServeFile(w, r, filePath)
		return
	}
	
	// Check for root index.html if requesting root
	if r.URL.Path == "/" {
		rootIndexPath, hasRootIndex := webUIFileMap["/"]
		if hasRootIndex {
			webUIFileMapMutex.RUnlock()
			// Add cache-busting headers
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, rootIndexPath)
			return
		}
	}
	
	webUIFileMapMutex.RUnlock()
	
	// Custom 404 response for everything else
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("File Not Found"))
}



// UpdateLoginJsonPrefix reads the web/login.json file, updates the apiPrefix
// field under the "app" section with the given prefix, and writes it back.
func UpdateLoginJsonPrefix(prefix string) error {
	loginJsonPath := "web/login.json"
	possiblePaths := []string{
		"web/login.json",
		"./web/login.json",
		"../web/login.json",
		"../../web/login.json",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			loginJsonPath = path
			break
		}
	}

	data, err := os.ReadFile(loginJsonPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", loginJsonPath, err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse %s: %w", loginJsonPath, err)
	}

	appSection, ok := config["app"].(map[string]interface{})
	if !ok {
		appSection = make(map[string]interface{})
		config["app"] = appSection
	}

	appSection["apiPrefix"] = prefix

	updatedData, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	if err := os.WriteFile(loginJsonPath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", loginJsonPath, err)
	}

	fmt.Println("Updated apiPrefix in", loginJsonPath, "to", prefix)
	return nil
}

// concat efficiently concatenates multiple strings using a bytes.Buffer.
// Returns an empty string if no arguments are provided.
func concat(strs ...string) string {
	buff := bytes.Buffer{}
	if strs != nil {
		for _, str := range strs {
			buff.WriteString(str)
		}
	}
	return buff.String()
}
