package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
)

type Item struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

var (
	mu     sync.RWMutex
	items  = []Item{{ID: 1, Name: "foo"}, {ID: 2, Name: "bar"}}
	nextID = 3
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

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /items", getItemsHandler)
	mux.HandleFunc("POST /items", createItemHandler)

	addr := ":8080"
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
