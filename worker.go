package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

type Query struct {
	ID     int    `json:"id,omitempty"`
	Prompt string `json:"prompt"`
}

type LLMResponse struct {
	ID   int    `json:"id,omitempty"`
	Text string `json:"text"`
}

func worker(workerID int, queries <-chan Query, apiKey string, wg *sync.WaitGroup) {
	defer wg.Done()
	for query := range queries {
		fmt.Printf("Worker %d processing query %d\n", workerID, query.ID)
		result, err := callOpenAI(query.Prompt, apiKey)
		if err != nil {
			result = fmt.Sprintf("Error: %v", err)
		}
		response := LLMResponse{ID: query.ID, Text: result}
		
		responseJSON, err := json.Marshal(response)
		if err != nil {
			log.Printf("Worker %d: error marshalling response: %v", workerID, err)
			continue
		}
		channelName := "query_result_" + fmt.Sprint(query.ID)
		err = rdb.Publish(ctx, channelName, string(responseJSON)).Err()
		if err != nil {
			log.Printf("Worker %d: error publishing result: %v", workerID, err)
		}
		err = rdb.Set(ctx, channelName, string(responseJSON), 0).Err()
		if err != nil {
			log.Printf("Worker %d: error setting result in Redis: %v", workerID, err)
		}
	}
}
