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
	items  = []Item{{ID: 1, Name: "The Best"}, {ID: 2, Name: "Software Engineer"}}
	nextID = 3
	apiKey string
	client serpapi.SerpApiClient
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
    results, err := executeSerpAPIRequest("synonyms for ok")
    if err != nil {
   		http.Error(w, "serpapi request failed", http.StatusBadGateway)
   		return
   	}

   organicResults, ok := results["organic_results"].([]interface{})
   if !ok {
        http.Error(w, "no organic results found", http.StatusNotFound)
        return
   }

   var targetResult map[string]interface{}
   for _, result := range organicResults {
        resultMap, ok := result.(map[string]interface{})
           if !ok {
               continue
           }

           // Check if this result has the target link
           if link, ok := resultMap["link"].(string); ok {
               // just chose this one for no particular reason
               if link == "https://www.merriam-webster.com/thesaurus/OK" {
                   targetResult = resultMap
                   break
               }
           }
       }

	if targetResult == nil {
		http.Error(w, "merriam-webster link not found", http.StatusNotFound)
		return
	}

	// Extract snippet_highlighted_words from the target result
	highlightedWordsArray, ok := targetResult["snippet_highlighted_words"].([]interface{})
	if !ok || len(highlightedWordsArray) == 0 {
		http.Error(w, "no highlighted words found", http.StatusNotFound)
		return
	}

	// Get the first element which is a comma-separated string
	firstElementStr, ok := highlightedWordsArray[0].(string)
	if !ok || firstElementStr == "" {
		http.Error(w, "no words in highlighted words array", http.StatusNotFound)
		return
	}

	// Split the comma-separated string into individual words
	words := strings.Split(firstElementStr, ",")

	// Trim whitespace from each word
	for i := range words {
		words[i] = strings.TrimSpace(words[i])
	}

	// Choose a random word from the list
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Return the random word
	writeJSON(w, http.StatusOK, words[rng.Intn(len(words))])
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

	results, err := executeSerpAPIRequest(itemName)
	if err != nil {
		http.Error(w, "serpapi request failed", http.StatusBadGateway)
		return
	}

	// Extract organic_results from the response
	organicResults, ok := results["organic_results"].([]interface{})
	if !ok || len(organicResults) == 0 {
		http.Error(w, "no organic results found", http.StatusNotFound)
		return
	}

	// Return the first result
	writeJSON(w, http.StatusOK, organicResults[0])
}

func executeSerpAPIRequest(searchFor string) (map[string]interface{}, error) {
	params := map[string]string{
		"q":             searchFor,
		"location":      "United States",
		"google_domain": "google.com",
		"hl":            "en",
		"gl":            "us",
	}

	results, err := client.Search(params)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /items", getItemsHandler)
	mux.HandleFunc("POST /items", createItemHandler)
	mux.HandleFunc("GET /findRandomItemFromList", findRandomItemFromList)

	apiKey = getSerpAPIKey()
	if apiKey == "" {
		log.Fatal("missing SERPAPI_API_KEY in environment or .env")
	}

	setting := serpapi.NewSerpApiClientSetting(apiKey)
	setting.Engine = "google"
	client = serpapi.NewClient(setting)

	addr := ":8080"
	log.Printf("Server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
