package main

import (
	"fmt"
	"sync"
)

type Query struct {
	ID     int    `json:"id,omitempty"`
	Prompt string `json:"prompt"`
}

type LLMResponse struct {
	ID   int    `json:"id"`
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
		responsesMap.Store(query.ID, LLMResponse{ID: query.ID, Text: result})
	}
}
