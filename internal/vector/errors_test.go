package vector_test

import (
	"errors"
	"testing"

	"github.com/partspile/site/internal/vector"
)

func TestIsEmbeddingUnavailable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"no embedded ads", vector.ErrNoEmbeddedAds, true},
		{"other", errors.New("database down"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := vector.IsEmbeddingUnavailable(tt.err); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
