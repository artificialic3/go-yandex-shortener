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

// generateKey генерирует случайный ключ заданной длины
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

	key, err := generateKey(4) // Используем 4 байта для ключа, что дает 8 символов в HEX.
	if err != nil {
		return "", err
	}

	// Добавить http:// не нужно, так как мы возвращаем ключ, который будет частью пути URL.
	shortURL := fmt.Sprintf("/%s", key)

	urlDatabase[key] = originalURL

	return shortURL, nil
}

func handleRequests(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		switch r.Method {
		case "GET":
			renderHTML(w)
		case "POST":
			shortenURL(w, r)
		default:
			http.Error(w, "Method not supported", http.StatusMethodNotAllowed)
		}
	} else {
		redirectURL(w, r)
	}
}

func shortenURL(w http.ResponseWriter, r *http.Request) {
	// Парсим форму, чтобы получить данные.
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Error parsing form data", http.StatusBadRequest)
		return
	}

	// Получаем значение URL из формы.
	originalURL := r.FormValue("url") // ключ 'url' соответствует атрибуту name инпута в HTML
	if originalURL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Добавляем схему, если она отсутствует.
	if !strings.HasPrefix(originalURL, "http://") && !strings.HasPrefix(originalURL, "https://") {
		originalURL = "http://" + originalURL
	}

	// Генерация и отправка короткого URL
	shortURL, err := generateShortURL(originalURL)
	if err != nil {
		http.Error(w, "Error generating short URL", http.StatusInternalServerError)
		return
	}

	// Заполнение заголовков для клиента с полным коротким URL.
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "http://localhost:8080%s", shortURL)
}

func redirectURL(w http.ResponseWriter, r *http.Request) {
	key := strings.TrimPrefix(r.URL.Path, "/")

	mu.RLock()
	originalURL, ok := urlDatabase[key]
	mu.RUnlock()

	if !ok {
		http.Error(w, "Short URL does not exist", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusTemporaryRedirect)
}

func renderHTML(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>URL Shortener Service</title>
</head>
<body>
    <h1>URL Shortener Service</h1>
    <form action="/" method="post" enctype="application/x-www-form-urlencoded">
        <input type="text" name="url" placeholder="Enter your URL here" size="50">
        <input type="submit" value="Shorten">
    </form>
</body>
</html>
`)
}

func main() {
	http.HandleFunc("/", handleRequests)

	log.Println("Starting URL Shortener server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
}
