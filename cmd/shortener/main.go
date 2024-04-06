package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

var urlDatabase = make(map[string]string)
var mu sync.RWMutex

func generateKey(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func generateShortURL(originalURL string) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	key, err := generateKey(4) // Generate a 4 byte key
	if err != nil {
		return "", err
	}
	shortURL := fmt.Sprintf("/%s", key)
	urlDatabase[key] = originalURL
	return shortURL, nil
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	originalURL := string(body)
	if originalURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	shortURL, err := generateShortURL(originalURL)
	if err != nil {
		http.Error(w, "Error generating short URL", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "http://localhost:8080%s", shortURL)
}

func redirectURL(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[1:]
	mu.RLock()
	originalURL, ok := urlDatabase[key]
	mu.RUnlock()

	if !ok {
		http.Error(w, "Short URL does not exist", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

func handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		shortenURL(w, r)
	} else if r.Method == http.MethodGet {
		redirectURL(w, r)
	} else {
		http.Error(w, "Only GET and POST methods are supported", http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/", handleRequests)

	log.Println("Starting server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
