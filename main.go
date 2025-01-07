package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const (
	base62Chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	shortIDLen  = 6
)

type URLStore struct {
	sync.RWMutex
	shortToLong map[string]string // shortID -> long URL
	longToShort map[string]string // long URL -> shortID
}

func newURLStore() *URLStore {
	return &URLStore{
		shortToLong: make(map[string]string),
		longToShort: make(map[string]string),
	}
}

func (s *URLStore) save(shortID, originalURL string) {
	s.Lock()
	defer s.Unlock()
	s.shortToLong[shortID] = originalURL
	s.longToShort[originalURL] = shortID
}

func (s *URLStore) getByShort(shortID string) (string, bool) {
	s.RLock()
	defer s.RUnlock()
	url, exists := s.shortToLong[shortID]
	return url, exists
}

func (s *URLStore) getByLong(originalURL string) (string, bool) {
	s.RLock()
	defer s.RUnlock()
	shortID, exists := s.longToShort[originalURL]
	return shortID, exists
}

var urlStore = newURLStore()

func generateShortID() string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, shortIDLen)
	for i := range b {
		b[i] = base62Chars[rand.Intn(len(base62Chars))]
	}
	return string(b)
}

func shortenURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		URL string `json:"url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Check if the URL already exists
	if shortID, exists := urlStore.getByLong(req.URL); exists {
		resp := map[string]string{"short_url": "http://localhost:8080/" + shortID}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Generate a new short URL if it doesn't exist
	shortID := generateShortID()
	urlStore.save(shortID, req.URL)

	resp := map[string]string{"short_url": "http://localhost:8080/" + shortID}
	json.NewEncoder(w).Encode(resp)
}

func resolveURLHandler(w http.ResponseWriter, r *http.Request) {
	shortID := r.URL.Path[1:]
	originalURL, exists := urlStore.getByShort(shortID)
	if !exists {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func main() {
	http.HandleFunc("/", resolveURLHandler)
	http.HandleFunc("/shorten", shortenURLHandler)

	log.Println("Server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
