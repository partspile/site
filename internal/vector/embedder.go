package vector

import (
	"fmt"
	"hash/fnv"
	"math"
	"strings"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

type Embedder interface {
	EmbedText(text string) ([]float32, error)
	EmbedTexts(texts []string) ([][]float32, error)
	EmbedQuery(text string) ([]float32, error)
	EmbedDocuments(texts []string) ([][]float32, error)
}

var activeEmbedder Embedder

func SetEmbedder(e Embedder) {
	activeEmbedder = e
}

// EmbedderInfo returns the Ollama provider identifier and model name.
func EmbedderInfo() (provider, model string) {
	return "ollama", config.OllamaEmbeddingModel
}

// InitEmbedder initializes the local Ollama embedding backend.
func InitEmbedder() error {
	if err := InitOllamaClient(); err != nil {
		return err
	}
	logger.Info("Embedder initialized", "provider", "ollama",
		"model", config.OllamaEmbeddingModel)
	return nil
}

func embedText(text string) ([]float32, error) {
	if activeEmbedder == nil {
		return nil, fmt.Errorf("embedder not initialized")
	}
	return activeEmbedder.EmbedText(text)
}

func embedTexts(texts []string) ([][]float32, error) {
	if activeEmbedder == nil {
		return nil, fmt.Errorf("embedder not initialized")
	}
	return activeEmbedder.EmbedTexts(texts)
}

func embedQuery(text string) ([]float32, error) {
	if activeEmbedder == nil {
		return nil, fmt.Errorf("embedder not initialized")
	}
	return activeEmbedder.EmbedQuery(text)
}

func embedDocuments(texts []string) ([][]float32, error) {
	if activeEmbedder == nil {
		return nil, fmt.Errorf("embedder not initialized")
	}
	return activeEmbedder.EmbedDocuments(texts)
}

type fakeEmbedder struct{}

func NewFakeEmbedder() Embedder {
	return fakeEmbedder{}
}

func (fakeEmbedder) EmbedText(text string) ([]float32, error) {
	vecs, err := fakeEmbedder{}.EmbedTexts([]string{text})
	if err != nil || len(vecs) == 0 {
		return nil, err
	}
	return vecs[0], nil
}

func (fakeEmbedder) EmbedTexts(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, fmt.Errorf("cannot embed empty text array")
	}
	out := make([][]float32, len(texts))
	for i, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("cannot embed empty text at index %d", i)
		}
		out[i] = hashEmbed(text)
	}
	return out, nil
}

func (f fakeEmbedder) EmbedQuery(text string) ([]float32, error) {
	return f.EmbedText(text)
}

func (f fakeEmbedder) EmbedDocuments(texts []string) ([][]float32, error) {
	return f.EmbedTexts(texts)
}

func hashEmbed(text string) []float32 {
	vec := make([]float32, config.OllamaEmbeddingDimensions)
	for _, word := range strings.Fields(strings.ToLower(text)) {
		h := fnv.New64a()
		_, _ = h.Write([]byte(word))
		x := h.Sum64()
		for j := 0; j < 8; j++ {
			idx := int(x % uint64(len(vec)))
			vec[idx] += 1
			x /= uint64(len(vec))
			if x == 0 {
				break
			}
		}
	}
	normalizeVector(vec)
	return vec
}

func cloneEmbedding(v []float32) []float32 {
	if len(v) == 0 {
		return nil
	}
	cp := make([]float32, len(v))
	copy(cp, v)
	return cp
}

func normalizeVector(vec []float32) {
	var sum float64
	for _, v := range vec {
		sum += float64(v) * float64(v)
	}
	if sum == 0 {
		vec[0] = 1
		return
	}
	scale := float32(1 / math.Sqrt(sum))
	for i := range vec {
		vec[i] *= scale
	}
}
