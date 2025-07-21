package handlers

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnyStringInSlice(t *testing.T) {
	tests := []struct {
		name     string
		sliceA   []string
		sliceB   []string
		expected bool
	}{
		{
			name:     "matching strings",
			sliceA:   []string{"apple", "banana", "cherry"},
			sliceB:   []string{"banana", "date"},
			expected: true,
		},
		{
			name:     "no matching strings",
			sliceA:   []string{"apple", "banana", "cherry"},
			sliceB:   []string{"date", "elderberry"},
			expected: false,
		},
		{
			name:     "empty slice A",
			sliceA:   []string{},
			sliceB:   []string{"apple", "banana"},
			expected: false,
		},
		{
			name:     "empty slice B",
			sliceA:   []string{"apple", "banana"},
			sliceB:   []string{},
			expected: false,
		},
		{
			name:     "both empty slices",
			sliceA:   []string{},
			sliceB:   []string{},
			expected: false,
		},
		{
			name:     "case sensitive matching",
			sliceA:   []string{"Apple", "Banana"},
			sliceB:   []string{"apple", "banana"},
			expected: false,
		},
		{
			name:     "multiple matches",
			sliceA:   []string{"apple", "banana", "cherry"},
			sliceB:   []string{"banana", "cherry", "date"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := anyStringInSlice(tt.sliceA, tt.sliceB)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHtmlEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no quotes",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "with double quotes",
			input:    `Hello "World"`,
			expected: `Hello \"World\"`,
		},
		{
			name:     "multiple quotes",
			input:    `"Hello" "World"`,
			expected: `\"Hello\" \"World\"`,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only quotes",
			input:    `""`,
			expected: `\"\"`,
		},
		{
			name:     "mixed content",
			input:    `Test "quoted" content with "multiple" quotes`,
			expected: `Test \"quoted\" content with \"multiple\" quotes`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := htmlEscape(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnyStringInSlice_Performance(t *testing.T) {
	// Test with larger slices to ensure performance is reasonable
	largeSliceA := make([]string, 1000)
	largeSliceB := make([]string, 1000)

	// Fill with unique values
	for i := 0; i < 1000; i++ {
		largeSliceA[i] = fmt.Sprintf("value_%d", i)
		largeSliceB[i] = fmt.Sprintf("other_%d", i)
	}

	// Add one matching value
	largeSliceA[500] = "match"
	largeSliceB[500] = "match"

	// This should find the match
	result := anyStringInSlice(largeSliceA, largeSliceB)
	assert.True(t, result)

	// Test with no matches
	largeSliceB[500] = "no_match"
	result = anyStringInSlice(largeSliceA, largeSliceB)
	assert.False(t, result)
}

func BenchmarkAnyStringInSlice(b *testing.B) {
	sliceA := []string{"apple", "banana", "cherry", "date", "elderberry"}
	sliceB := []string{"fig", "grape", "honeydew", "kiwi", "lemon"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		anyStringInSlice(sliceA, sliceB)
	}
}

func BenchmarkHtmlEscape(b *testing.B) {
	testString := `This is a "test" string with "multiple" quotes and "some" more "quotes"`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		htmlEscape(testString)
	}
}
