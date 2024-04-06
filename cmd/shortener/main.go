package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

var urlDatabase = make(map[string]string)
var mu sync.RWMutex

func generateKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateShortURL(originalURL string) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	key, err := generateKey(4)
	if err != nil {
		return "", err
	}
	shortURL := fmt.Sprintf("/%s", key)
	urlDatabase[key] = originalURL
	return shortURL, nil
}

func handleRequests(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

	switch r.Method {
	case "GET":
		redirectURL(w, r)
	case "POST":
		shortenURL(w, r)
	default:
		http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
	}
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	originalURL := r.URL.Query().Get("url")
	if originalURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		originalURL = "http://" + originalURL
	}
	shortURL, err := generateShortURL(originalURL)
	if err != nil {
		log.Printf("Error generating short URL: %v", err)
		http.Error(w, "Error generating short URL", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "http://localhost:8080%s", shortURL)
}

func redirectURL(w http.ResponseWriter, r *http.Request) {
	// Extract the key from the URL path, discarding the '/' prefix
	key := strings.TrimPrefix(r.URL.Path, "/")
	if key == "" {
		http.Error(w, "Missing key to redirect", http.StatusBadRequest)
		return
	}

	// Lock the mutex for reading and defer the unlock
	mu.RLock()
	originalURL, ok := urlDatabase[key]
	mu.RUnlock()

	// Check if the original URL was found in our 'database'
	if !ok {
		http.Error(w, "Short URL does not exist", http.StatusNotFound)
		return
	}

	// Check for a valid URL format
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		log.Printf("Invalid URL format: %s", originalURL)
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	// Redirect to the original URL
	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

func main() {
	// Attach the handler function to handle all HTTP requests
	http.HandleFunc("/", handleRequests)

	// Start the HTTP server and log to console
	log.Println("Starting URL Shortener server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}
