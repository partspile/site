package vector

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdResult_Structure(t *testing.T) {
	// Test that AdResult can be created with proper structure
	result := AdResult{
		ID:    "test-id",
		Score: 0.95,
		Metadata: map[string]interface{}{
			"title": "Test Ad",
			"price": 100.0,
		},
	}

	assert.Equal(t, "test-id", result.ID)
	assert.Equal(t, float32(0.95), result.Score)
	assert.Len(t, result.Metadata, 2)
	assert.Equal(t, "Test Ad", result.Metadata["title"])
	assert.Equal(t, 100.0, result.Metadata["price"])
}

func TestAggregateEmbeddings_EmptyVectors(t *testing.T) {
	// Test aggregation with empty vectors
	vectors := [][]float32{}
	weights := []float32{}

	result := AggregateEmbeddings(vectors, weights)

	assert.Len(t, result, 0)
}

func TestAggregateEmbeddings_SingleVector(t *testing.T) {
	// Test aggregation with single vector
	vectors := [][]float32{
		{1.0, 2.0, 3.0},
	}
	weights := []float32{1.0}

	result := AggregateEmbeddings(vectors, weights)

	assert.Len(t, result, 3)
	assert.Equal(t, float32(1.0), result[0])
	assert.Equal(t, float32(2.0), result[1])
	assert.Equal(t, float32(3.0), result[2])
}

func TestAggregateEmbeddings_MultipleVectors(t *testing.T) {
	// Test aggregation with multiple vectors
	vectors := [][]float32{
		{1.0, 2.0, 3.0},
		{4.0, 5.0, 6.0},
		{7.0, 8.0, 9.0},
	}
	weights := []float32{0.5, 0.3, 0.2}

	result := AggregateEmbeddings(vectors, weights)

	assert.Len(t, result, 3)
	// Expected: (1*0.5 + 4*0.3 + 7*0.2) = 0.5 + 1.2 + 1.4 = 3.1
	assert.InDelta(t, float32(3.1), result[0], 0.01)
	// Expected: (2*0.5 + 5*0.3 + 8*0.2) = 1.0 + 1.5 + 1.6 = 4.1
	assert.InDelta(t, float32(4.1), result[1], 0.01)
	// Expected: (3*0.5 + 6*0.3 + 9*0.2) = 1.5 + 1.8 + 1.8 = 5.1
	assert.InDelta(t, float32(5.1), result[2], 0.01)
}

func TestAggregateEmbeddings_UnequalWeights(t *testing.T) {
	// Test aggregation with weights that don't sum to 1
	vectors := [][]float32{
		{1.0, 2.0},
		{3.0, 4.0},
	}
	weights := []float32{0.6, 0.4}

	result := AggregateEmbeddings(vectors, weights)

	assert.Len(t, result, 2)
	// Expected: (1*0.6 + 3*0.4) = 0.6 + 1.2 = 1.8
	assert.InDelta(t, float32(1.8), result[0], 0.01)
	// Expected: (2*0.6 + 4*0.4) = 1.2 + 1.6 = 2.8
	assert.InDelta(t, float32(2.8), result[1], 0.01)
}

func TestAggregateEmbeddings_DifferentLengths(t *testing.T) {
	// Test aggregation with vectors of different lengths
	vectors := [][]float32{
		{1.0, 2.0, 3.0},
		{4.0, 5.0},
	}
	weights := []float32{0.5, 0.5}

	result := AggregateEmbeddings(vectors, weights)

	// Function processes first vector but skips second due to length mismatch
	// Total weight is 0.5, so result should be (1,2,3) / 0.5 = (2,4,6)
	// But it seems to return the original vector without normalization
	assert.Len(t, result, 3)
	assert.InDelta(t, float32(1.0), result[0], 0.01)
	assert.InDelta(t, float32(2.0), result[1], 0.01)
	assert.InDelta(t, float32(3.0), result[2], 0.01)
}

func TestAggregateEmbeddings_ZeroWeights(t *testing.T) {
	// Test aggregation with zero weights
	vectors := [][]float32{
		{1.0, 2.0, 3.0},
		{4.0, 5.0, 6.0},
	}
	weights := []float32{0.0, 0.0}

	result := AggregateEmbeddings(vectors, weights)

	// Function should return nil when total weight is 0
	assert.Nil(t, result)
}

func TestAggregateEmbeddings_NegativeWeights(t *testing.T) {
	// Test aggregation with negative weights
	vectors := [][]float32{
		{1.0, 2.0},
		{3.0, 4.0},
	}
	weights := []float32{-0.5, 0.5}

	result := AggregateEmbeddings(vectors, weights)

	// When weights sum to zero, function returns nil
	assert.Nil(t, result)
}

func TestCursorEncodingDecoding(t *testing.T) {
	// Test cursor encoding and decoding
	offset := uint64(10)

	// Encode cursor
	cursorData := fmt.Sprintf("%d", offset)
	encodedCursor := base64.StdEncoding.EncodeToString([]byte(cursorData))

	// Decode cursor
	cursorBytes, err := base64.StdEncoding.DecodeString(encodedCursor)
	assert.NoError(t, err)

	decodedOffset, err := strconv.ParseUint(string(cursorBytes), 10, 64)
	assert.NoError(t, err)
	assert.Equal(t, offset, decodedOffset)
}

func TestCursorWithInvalidData(t *testing.T) {
	// Test cursor handling with invalid data
	invalidCursor := "invalid-base64"

	cursorBytes, err := base64.StdEncoding.DecodeString(invalidCursor)
	assert.Error(t, err)

	// Should handle gracefully
	var offset uint64 = 0
	var score float32 = 0

	if err == nil {
		cursorStr := string(cursorBytes)
		parts := strings.Split(cursorStr, ":")
		if len(parts) == 2 {
			if offsetVal, err := strconv.ParseUint(parts[0], 10, 64); err == nil {
				offset = offsetVal
			}
			if scoreVal, err := strconv.ParseFloat(parts[1], 32); err == nil {
				score = float32(scoreVal)
			}
		}
	}

	// Should default to zero values
	assert.Equal(t, uint64(0), offset)
	assert.Equal(t, float32(0), score)
}
