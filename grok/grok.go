package grok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	GrokAPIURL = "https://api.x.ai/v1/chat/completions"
	GrokModel  = "grok-3-mini"
)

type GrokRequest struct {
	Model           string        `json:"model"`
	ReasoningEffort string        `json:"reasoning_effort"`
	Messages        []GrokMessage `json:"messages"`
}

type GrokMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GrokResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// CallGrok sends prompts to the Grok API and returns the response string
func CallGrok(systemPrompt, userPrompt string) (string, error) {
	apiKey := os.Getenv("GROK_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GROK_API_KEY environment variable not set")
	}

	payload := GrokRequest{
		Model:           GrokModel,
		ReasoningEffort: "low",
		Messages: []GrokMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	data, err := json.MarshalIndent(payload, "", "\t")
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	fmt.Println("REQUEST")
	fmt.Println(string(data))

	req, err := http.NewRequest("POST", GrokAPIURL, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Grok API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Grok API returned status %d: %s", resp.StatusCode, string(body))
	}

	var grokResp GrokResponse
	err = json.NewDecoder(resp.Body).Decode(&grokResp)
	if err != nil {
		return "", fmt.Errorf("failed to decode Grok response: %w", err)
	}

	if len(grokResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Grok API")
	}

	fmt.Println("RESPONSE")
	fmt.Println(grokResp.Choices[0].Message.Content)

	return grokResp.Choices[0].Message.Content, nil
}
