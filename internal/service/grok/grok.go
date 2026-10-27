package grok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

type GrokRequest struct {
	Model           string        `json:"model"`
	ReasoningEffort string        `json:"reasoning_effort"`
	Messages        []GrokMessage `json:"messages"`
}

// GrokMessage is a chat message. Content is either a string or []ContentPart.
type GrokMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type ContentPart struct {
	Type     string    `json:"type"`
	Text     string    `json:"text,omitempty"`
	ImageURL *ImageURL `json:"image_url,omitempty"`
}

type ImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
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
	return CallGrokConv(systemPrompt, userPrompt, "")
}

// CallGrokConv is like CallGrok but sets the x-grok-conv-id header so
// requests sharing a conv ID route to the same server, maximizing prompt
// cache hits on a constant system prompt.
func CallGrokConv(systemPrompt, userPrompt, convID string) (string, error) {
	return CallGrokConvParts(systemPrompt, []ContentPart{
		{Type: "text", Text: userPrompt},
	}, convID)
}

// CallGrokConvParts sends a multimodal user message (text + images).
func CallGrokConvParts(systemPrompt string, parts []ContentPart,
	convID string) (string, error) {
	apiKey := config.GrokAPIKey
	if apiKey == "" {
		return "", fmt.Errorf("GROK_API_KEY environment variable not set")
	}

	payload := GrokRequest{
		Model:           config.GrokModel,
		ReasoningEffort: config.GrokReasoningEffort,
		Messages: []GrokMessage{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role:    "user",
				Content: parts,
			},
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	logger.Debug("Grok API request",
		"systemPrompt", systemPrompt, "parts", len(parts))

	req, err := http.NewRequest("POST", config.GrokAPIURL, bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if convID != "" {
		req.Header.Set("x-grok-conv-id", convID)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Grok API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Grok API returned status %d: %s",
			resp.StatusCode, string(body))
	}

	var grokResp GrokResponse
	err = json.NewDecoder(resp.Body).Decode(&grokResp)
	if err != nil {
		return "", fmt.Errorf("failed to decode Grok response: %w", err)
	}

	if len(grokResp.Choices) == 0 {
		return "", fmt.Errorf("no response from Grok API")
	}

	return grokResp.Choices[0].Message.Content, nil
}
