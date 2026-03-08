package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	serpapi "github.com/serpapi/serpapi-golang"
)

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var (
	mu     sync.RWMutex
	items  = []Item{{ID: 1, Name: "foo"}, {ID: 2, Name: "bar"}}
	nextID = 3
	apiKey = getSerpAPIKey()
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func getItemsHandler(w http.ResponseWriter, r *http.Request) {
	mu.RLock()
	snapshot := make([]Item, len(items))
	copy(snapshot, items)
	mu.RUnlock()
	writeJSON(w, http.StatusOK, snapshot)
}

func createItemHandler(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	mu.Lock()
	item.ID = nextID
	nextID++
	items = append(items, item)
	mu.Unlock()
	writeJSON(w, http.StatusCreated, item)
}

func getSerpAPIKey() string {
	if value, ok := os.LookupEnv("SERPAPI_API_KEY"); ok && value != "" {
		return value
	}

	data, err := os.ReadFile(".env")
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "SERPAPI_API_KEY=") {
			return strings.Trim(strings.TrimPrefix(line, "SERPAPI_API_KEY="), "\"'")
		}
	}

	return ""
}

func findRandomItemFromList(w http.ResponseWriter, r *http.Request) {
	if apiKey == "" {
		http.Error(w, "missing SERPAPI_API_KEY in environment or .env", http.StatusInternalServerError)
		return
	}

	mu.RLock()
	if len(items) == 0 {
		mu.RUnlock()
		http.Error(w, "no items available", http.StatusNotFound)
		return
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	itemName := items[rng.Intn(len(items))].Name
	mu.RUnlock()

	setting := serpapi.NewSerpApiClientSetting(apiKey)
	setting.Engine = "google"
	client := serpapi.NewClient(setting)

	params := map[string]string{
		"q":             itemName,
		"location":      "United States",
		"google_domain": "google.com",
		"hl":            "en",
		"gl":            "us",
	}

	results, err := client.Search(params)
	if err != nil {
		http.Error(w, "serpapi request failed", http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /items", getItemsHandler)
	mux.HandleFunc("POST /items", createItemHandler)
	mux.HandleFunc("GET /findRandomItemFromList", findRandomItemFromList)

	addr := ":8080"
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
