// Package main provides the entry point for the Gardener application.
// It initializes the database, configures HTTP routes, serves the frontend,
// and launches the local web server.
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"gardener/internal/api"
	"gardener/internal/db"
)

// frontendFS contains the embedded frontend static assets.
// The //go:embed directive bundles the contents of frontend/dist/
// into the binary at compile time, enabling single-file deployment.
//
//go:embed frontend/dist/*
var frontendFS embed.FS

// main initializes the application:
//   - Sets up the SQLite database via db.Init()
//   - Configures HTTP handlers for API endpoints and static files
//   - Starts the HTTP server on 127.0.0.1:8080
//   - Automatically opens the default web browser to the application URL
//
// The function exits via log.Fatal if the server fails to start.
func main() {
	if err := db.Init(); err != nil {
		log.Fatalf("❌ DB init failed: %v", err)
	}

	mux := http.NewServeMux()

	// Register API handlers for plant management.
	// Supported methods: GET (list), POST (create/update), DELETE (remove).
	mux.HandleFunc("/api/plants", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			api.GetPlants(w, r)
		case http.MethodPost:
			api.SavePlant(w, r)
		case http.MethodDelete:
			api.DeletePlant(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Register handler for serving plant photos.
	mux.HandleFunc("/api/photos/", api.GetPhoto)

	// Serve embedded frontend static files.
	// fs.Sub creates a sub-filesystem rooted at frontend/dist/.
	sub, _ := fs.Sub(frontendFS, "frontend/dist")
	mux.Handle("/", http.FileServer(http.FS(sub)))

	addr := "127.0.0.1:8080"
	log.Printf("🌱 Огородник запущен: http://%s", addr)

	// Launch default browser in a separate goroutine to avoid blocking server startup.
	go openBrowser("http://" + addr)

	// Start HTTP server; log.Fatal exits if ListenAndServe fails.
	log.Fatal(http.ListenAndServe(addr, mux))
}

// openBrowser attempts to open the given URL in the system's default web browser.
// It selects the appropriate command based on the runtime OS:
//   - Windows: "cmd /c start"
//   - macOS: "open"
//   - Linux/other: "xdg-open"
//
// Errors during browser launch are logged as warnings but do not stop the application.
func openBrowser(url string) {
	var browserCmd string
	switch runtime.GOOS {
	case "windows":
		browserCmd = "cmd /c start"
	case "darwin":
		browserCmd = "open"
	default:
		browserCmd = "xdg-open"
	}
	parts := strings.Fields(browserCmd)

	cmd := exec.Command(parts[0], append(parts[1:], url)...)
	if err := cmd.Start(); err != nil {
		log.Printf("⚠️ Failed to open browser: %v", err)
	}
}
