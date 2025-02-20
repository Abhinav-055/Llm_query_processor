package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"github.com/joho/godotenv"
)

var (
	queries      chan Query      
	responsesMap sync.Map         
	idCounter    int32           
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, ensure OPENAI_API_KEY is set in your environment.")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set. Please set it in your .env file or environment.")
	}

	queries = make(chan Query, 10)

	var wg sync.WaitGroup
	numWorkers := 3
	for i := 1; i <= numWorkers; i++ {
		wg.Add(1)
		go worker(i, queries, apiKey, &wg)
	}

	http.HandleFunc("/query", queryPostHandler) 
	http.HandleFunc("/result", queryGetHandler) 

	port := "8080"
	fmt.Printf("Server running on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
func queryPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Query
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	newID := int(atomic.AddInt32(&idCounter, 1))
	req.ID = newID

	responsesMap.Store(newID, LLMResponse{ID: newID, Text: "Processing..."})
	queries <- req

	resp := map[string]interface{}{
		"message": "Query received",
		"id":      newID,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func queryGetHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Only GET is allowed", http.StatusMethodNotAllowed)
        return
    }

    idStr := r.URL.Query().Get("id")
    if idStr == "" {
        http.Error(w, "Missing id parameter", http.StatusBadRequest)
        return
    }

    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid id parameter", http.StatusBadRequest)
        return
    }

    value, ok := responsesMap.Load(id)
    if !ok {
        log.Printf("No query found with id %d", id)
        http.Error(w, "No query found with that id", http.StatusNotFound)
        return
    }
    
    log.Printf("Retrieved response for id %d: %+v", id, value)

    response, ok := value.(LLMResponse)
    if !ok {
        log.Printf("Type assertion failed for id %d: got %T", id, value)
        http.Error(w, "Error processing response: unexpected data format", http.StatusInternalServerError)
        return
    }

    if response.Text == "Processing..." {
        log.Printf("Query id %d is still processing", id)
    }

    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(response); err != nil {
        log.Printf("Error encoding JSON response for id %d: %v", id, err)
    }
}

