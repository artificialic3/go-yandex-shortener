package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

var (
	urlDatabase = make(map[string]string)
	mutex       sync.RWMutex
)

func generateKey(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func shortenURL(originalURL string) (string, error) {
	mutex.Lock()
	defer mutex.Unlock()

	key, err := generateKey(4)
	if err != nil {
		return "", err
	}
	urlDatabase[key] = originalURL
	return key, nil
}

func handleRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		originalURL := string(body)
		if originalURL == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		key, err := shortenURL(originalURL)
		if err != nil {
			http.Error(w, "Error generating short URL", http.StatusInternalServerError)
			return
		}

		shortURL := fmt.Sprintf("http://%s/%s", r.Host, key)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, shortURL)

	case http.MethodGet:
		key := r.URL.Path[1:]

		mutex.RLock()
		originalURL, ok := urlDatabase[key]
		mutex.RUnlock()

		if !ok {
			http.Error(w, "Short URL does not exist", http.StatusBadRequest)
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/", handleRequests)

	log.Println("Server starting on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
