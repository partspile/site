package vector

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/parts-pile/site/config"
	genai "google.golang.org/genai"
)

var (
	geminiClient *genai.Client
)

// InitGeminiClient initializes the Gemini embedding client
func InitGeminiClient() error {
	apiKey := config.GeminiAPIKey
	if apiKey == "" {
		return fmt.Errorf("missing Gemini API key")
	}
	// The client gets the API key from the environment variable `GEMINI_API_KEY`
	client, err := genai.NewClient(context.Background(), nil)
	if err != nil {
		return err
	}
	geminiClient = client
	return nil
}

// EmbedText generates an embedding for the given text using Gemini
func EmbedText(text string) ([]float32, error) {
	embeddings, err := EmbedTexts([]string{text})
	if err != nil {
		return nil, err
	}
	return embeddings[0], nil
}

// EmbedTexts generates embeddings for multiple texts using Gemini in a single API call
func EmbedTexts(texts []string) ([][]float32, error) {
	if geminiClient == nil {
		return nil, fmt.Errorf("Gemini client not initialized")
	}

	if len(texts) == 0 {
		return nil, fmt.Errorf("cannot embed empty text array")
	}

	// Prepare content array for batch processing
	var contents []*genai.Content
	for i, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("cannot embed empty text at index %d", i)
		}
		contents = append(contents, genai.Text(text)...)
	}

	log.Printf("[embedding] Calculating batch embedding vectors for %d texts", len(texts))
	ctx := context.Background()
	dimensions := int32(config.GeminiEmbeddingDimensions)
	resp, err := geminiClient.Models.EmbedContent(ctx, config.GeminiEmbeddingModel, contents,
		&genai.EmbedContentConfig{OutputDimensionality: &dimensions})
	if err != nil {
		return nil, fmt.Errorf("Gemini batch embedding API error: %w", err)
	}
	if resp == nil || len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned from Gemini API")
	}
	if len(resp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("mismatch between requested texts (%d) and returned embeddings (%d)", len(texts), len(resp.Embeddings))
	}

	// Extract embeddings from response
	var embeddings [][]float32
	for i, embedding := range resp.Embeddings {
		if embedding == nil {
			return nil, fmt.Errorf("nil embedding returned for text at index %d", i)
		}
		embeddings = append(embeddings, embedding.Values)
	}

	log.Printf("[embedding] Successfully generated %d embeddings in batch", len(embeddings))
	return embeddings, nil
}
