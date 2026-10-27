package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/partspile/site/internal/config"
)

const (
	ollamaPrefixQuery    = "search_query: "
	ollamaPrefixDocument = "search_document: "
	ollamaEmbedTimeout   = 60 * time.Second
)

type ollamaEmbedder struct {
	baseURL string
	model   string
	client  *http.Client
}

type ollamaEmbedRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type ollamaEmbedResponse struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func InitOllamaClient() error {
	baseURL := normalizeOllamaBaseURL(config.OllamaURL)
	SetEmbedder(ollamaEmbedder{
		baseURL: baseURL,
		model:   config.OllamaEmbeddingModel,
		client:  &http.Client{Timeout: ollamaEmbedTimeout},
	})
	return nil
}

// normalizeOllamaBaseURL accepts full URLs (http://localhost:11434) or
// host:port values from Render private networking (ollama-a1b2:11434).
func normalizeOllamaBaseURL(url string) string {
	base := strings.TrimRight(url, "/")
	if base != "" && !strings.Contains(base, "://") {
		base = "http://" + base
	}
	return base
}

func (o ollamaEmbedder) EmbedText(text string) ([]float32, error) {
	vecs, err := o.embed([]string{text}, ollamaPrefixDocument)
	if err != nil || len(vecs) == 0 {
		return nil, err
	}
	return vecs[0], nil
}

func (o ollamaEmbedder) EmbedTexts(texts []string) ([][]float32, error) {
	return o.embed(texts, ollamaPrefixDocument)
}

func (o ollamaEmbedder) EmbedQuery(text string) ([]float32, error) {
	vecs, err := o.embed([]string{text}, ollamaPrefixQuery)
	if err != nil || len(vecs) == 0 {
		return nil, err
	}
	return vecs[0], nil
}

func (o ollamaEmbedder) EmbedDocuments(texts []string) ([][]float32, error) {
	return o.embed(texts, ollamaPrefixDocument)
}

func (o ollamaEmbedder) embed(texts []string, prefix string) ([][]float32, error) {
	if o.client == nil {
		return nil, fmt.Errorf("Ollama client not initialized")
	}
	if len(texts) == 0 {
		return nil, fmt.Errorf("cannot embed empty text array")
	}
	inputs := make([]string, len(texts))
	for i, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("cannot embed empty text at index %d", i)
		}
		inputs[i] = prefix + text
	}

	payload := ollamaEmbedRequest{Model: o.model, Input: inputs}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", o.baseURL+"/api/embed",
		bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API returned status %d: %s",
			resp.StatusCode, string(body))
	}

	var embedResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}
	if len(embedResp.Embeddings) != len(texts) {
		return nil, fmt.Errorf("unexpected Ollama embedding response: got %d, want %d",
			len(embedResp.Embeddings), len(texts))
	}

	out := make([][]float32, len(texts))
	for i, emb := range embedResp.Embeddings {
		if len(emb) == 0 {
			return nil, fmt.Errorf("empty embedding at index %d", i)
		}
		out[i] = emb
	}
	return out, nil
}
