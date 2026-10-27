package vector_test

import (
	"testing"

	"github.com/partspile/site/internal/vector"
)

func TestAggregateEmbeddings(t *testing.T) {
	vecs := [][]float32{{1, 0}, {0, 1}}
	weights := []float32{1, 1}
	got := vector.AggregateEmbeddings(vecs, weights)
	if got == nil {
		t.Fatal("expected non-nil result")
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0] != 0.5 || got[1] != 0.5 {
		t.Fatalf("got %v", got)
	}
}

func TestAggregateEmbeddingsEmpty(t *testing.T) {
	if got := vector.AggregateEmbeddings(nil, nil); got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}
