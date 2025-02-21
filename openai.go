package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIRequest struct {
	Model    string          `json:"model"`
	Store    bool            `json:"store"`
	Messages []OpenAIMessage `json:"messages"`
}

type OpenAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message OpenAIMessage `json:"message"`
	} `json:"choices"`
}

func callOpenAI(prompt string, apiKey string) (string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	requestPayload := OpenAIRequest{
		Model: "gpt-4o-mini",
		Store: true,
		Messages: []OpenAIMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	fmt.Printf("callOpenAI: Request took %v\n", time.Since(start))
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Println("Raw Response:", string(respBody))

	var apiResp OpenAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", err
	}

	if len(apiResp.Choices) > 0 {
		return apiResp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("no response choices received")
}
