package main

import (
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAdImage(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		width    int
		height   int
		adID     int
		imageNum int
	}{
		{
			name:     "basic image generation",
			title:    "Test Ad Title",
			width:    800,
			height:   600,
			adID:     123,
			imageNum: 1,
		},
		{
			name:     "small image",
			title:    "Small Ad",
			width:    200,
			height:   150,
			adID:     456,
			imageNum: 2,
		},
		{
			name:     "large image",
			title:    "Large Ad Title",
			width:    1920,
			height:   1080,
			adID:     789,
			imageNum: 3,
		},
		{
			name:     "empty title",
			title:    "",
			width:    400,
			height:   300,
			adID:     999,
			imageNum: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := generateAdImage(tt.title, tt.width, tt.height, tt.adID, tt.imageNum)

			require.NoError(t, err)
			assert.NotNil(t, buf)
			assert.Greater(t, buf.Len(), 0)

			// Verify the buffer contains valid PNG data
			img, err := png.Decode(buf)
			require.NoError(t, err)

			// Check image dimensions
			bounds := img.Bounds()
			assert.Equal(t, tt.width, bounds.Dx())
			assert.Equal(t, tt.height, bounds.Dy())

			// Verify it's an RGBA image
			_, ok := img.(*image.RGBA)
			assert.True(t, ok, "Image should be RGBA")
		})
	}
}

func TestAddText(t *testing.T) {
	// Create a test image
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))

	// Test adding text
	text := "Test Text\nMultiple Lines"
	textColor := image.NewUniform(color.RGBA{255, 255, 255, 255})

	// This should not panic
	addText(img, text, textColor, 200, 150)

	// Verify the image was modified (has some non-zero pixels)
	hasNonZero := false
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if r > 0 || g > 0 || b > 0 || a > 0 {
				hasNonZero = true
				break
			}
		}
		if hasNonZero {
			break
		}
	}

	assert.True(t, hasNonZero, "Text should have been drawn on the image")
}

func TestGenerateAdImage_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		width       int
		height      int
		adID        int
		imageNum    int
		expectError bool
	}{
		{
			name:        "zero dimensions",
			title:       "Test",
			width:       0,
			height:      0,
			adID:        1,
			imageNum:    1,
			expectError: true,
		},
		{
			name:        "negative dimensions",
			title:       "Test",
			width:       -100,
			height:      -100,
			adID:        1,
			imageNum:    1,
			expectError: false, // The function doesn't actually check for negative dimensions
		},
		{
			name:        "very large dimensions",
			title:       "Test",
			width:       10000,
			height:      10000,
			adID:        1,
			imageNum:    1,
			expectError: false, // Should work but be slow
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, err := generateAdImage(tt.title, tt.width, tt.height, tt.adID, tt.imageNum)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, buf)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, buf)
			}
		})
	}
}

func TestGenerateAdImage_Consistency(t *testing.T) {
	// Test that generating the same image multiple times produces valid results
	// Note: Images won't be identical due to random background colors
	title := "Consistency Test"
	width, height := 400, 300
	adID, imageNum := 123, 1

	buf1, err1 := generateAdImage(title, width, height, adID, imageNum)
	require.NoError(t, err1)

	buf2, err2 := generateAdImage(title, width, height, adID, imageNum)
	require.NoError(t, err2)

	// Both images should be valid PNGs with reasonable sizes
	assert.Greater(t, buf1.Len(), 1000)
	assert.Greater(t, buf2.Len(), 1000)

	// Decode both images to verify they're valid
	img1, err := png.Decode(buf1)
	require.NoError(t, err)

	img2, err := png.Decode(buf2)
	require.NoError(t, err)

	// Both should have the same dimensions
	bounds1 := img1.Bounds()
	bounds2 := img2.Bounds()
	assert.Equal(t, bounds1, bounds2)
	assert.Equal(t, width, bounds1.Dx())
	assert.Equal(t, height, bounds1.Dy())
}

func BenchmarkGenerateAdImage(b *testing.B) {
	title := "Benchmark Test Ad Title"
	width, height := 800, 600
	adID, imageNum := 123, 1

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, err := generateAdImage(title, width, height, adID, imageNum)
		if err != nil {
			b.Fatal(err)
		}
		if buf == nil {
			b.Fatal("Generated buffer is nil")
		}
	}
}
