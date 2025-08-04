package vector

import (
	"testing"
)

func TestQuerySimilarAds_ThresholdParameter(t *testing.T) {
	// This test verifies that the threshold parameter is properly accepted
	// by the QuerySimilarAds function signature

	// Create a dummy embedding
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}

	// Test that the function signature accepts the threshold parameter
	// We can't actually test the Qdrant query without a real connection,
	// but we can verify the function signature is correct
	_, _, err := QuerySimilarAds(embedding, 10, "", 0.7)

	// The error should be about Qdrant client not being initialized,
	// not about the function signature
	if err == nil {
		t.Error("Expected error about Qdrant client not being initialized")
	}

	// Test with different threshold values
	_, _, err = QuerySimilarAds(embedding, 10, "", 0.5)
	if err == nil {
		t.Error("Expected error about Qdrant client not being initialized")
	}

	_, _, err = QuerySimilarAds(embedding, 10, "", 0.9)
	if err == nil {
		t.Error("Expected error about Qdrant client not being initialized")
	}
}

func TestQuerySimilarAdsWithFilter_ThresholdParameter(t *testing.T) {
	// This test verifies that the threshold parameter is properly accepted
	// by the QuerySimilarAdsWithFilter function signature

	// Create a dummy embedding
	embedding := make([]float32, 768)
	for i := range embedding {
		embedding[i] = 0.1
	}

	// Test that the function signature accepts the threshold parameter
	// We can't actually test the Qdrant query without a real connection,
	// but we can verify the function signature is correct
	_, _, err := QuerySimilarAdsWithFilter(embedding, nil, 10, "", 0.7)

	// The error should be about Qdrant client not being initialized,
	// not about the function signature
	if err == nil {
		t.Error("Expected error about Qdrant client not being initialized")
	}

	// Test with different threshold values
	_, _, err = QuerySimilarAdsWithFilter(embedding, nil, 10, "", 0.5)
	if err == nil {
		t.Error("Expected error about Qdrant client not being initialized")
	}

	_, _, err = QuerySimilarAdsWithFilter(embedding, nil, 10, "", 0.9)
	if err == nil {
		t.Error("Expected error about Qdrant client not being initialized")
	}
}
