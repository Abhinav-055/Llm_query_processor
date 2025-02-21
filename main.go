package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
)

var (
	queries   chan Query
	idCounter int32
	ctx       = context.Background()
	rdb       *redis.Client
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, ensure OPENAI_API_KEY is set in your environment.")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY is not set. Please set it in your .env file or environment.")
	}

	rdb = redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Could not connect to Redis: %v", err)
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

	channelName := "query_result_" + idStr
	fmt.Printf("GET handler: Subscribing to channel %s\n", channelName)
	sub := rdb.Subscribe(ctx, channelName)
	defer sub.Close()
    
	storedResult, err := rdb.Get(ctx, channelName).Result()
	if err == nil && storedResult != "" {
		
		var innerResp LLMResponse
		if err := json.Unmarshal([]byte(storedResult), &innerResp); err != nil {
			innerResp.Text = storedResult
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(innerResp)
		return
	}

	ch := sub.Channel()
	var rawResult string
	start := time.Now()
	select {
	case msg := <-ch:
		rawResult = msg.Payload
		fmt.Printf("GET handler: Received message in %v\n", time.Since(start))
	case <-time.After(15 * time.Second):
		http.Error(w, "Timeout waiting for result", http.StatusRequestTimeout)
		return
	}
	var response LLMResponse
	if err := json.Unmarshal([]byte(rawResult), &response); err != nil {
		response = LLMResponse{
			Text: rawResult,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
