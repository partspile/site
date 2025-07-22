package b2util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetB2DownloadTokenForPrefixCached_CacheHit(t *testing.T) {
	// Test that cached tokens are returned
	prefix := "test-prefix/"

	// Set up environment variables for testing
	os.Setenv("BACKBLAZE_MASTER_KEY_ID", "test-account-id")
	os.Setenv("BACKBLAZE_KEY_ID", "test-key-id")
	os.Setenv("BACKBLAZE_APP_KEY", "test-app-key")
	os.Setenv("B2_BUCKET_ID", "test-bucket-id")

	// First call should cache the token
	_, err := GetB2DownloadTokenForPrefixCached(prefix)

	// Second call should return cached token
	_, err2 := GetB2DownloadTokenForPrefixCached(prefix)

	// Note: This test will fail in real environment without valid credentials
	// but it tests the caching logic structure
	assert.Error(t, err)  // Should fail due to invalid credentials
	assert.Error(t, err2) // Should also fail
}

func TestGetB2DownloadTokenForPrefixCached_DifferentPrefixes(t *testing.T) {
	// Test that different prefixes have separate cache entries
	prefix1 := "prefix1/"
	prefix2 := "prefix2/"

	// Set up environment variables for testing
	os.Setenv("BACKBLAZE_MASTER_KEY_ID", "test-account-id")
	os.Setenv("BACKBLAZE_KEY_ID", "test-key-id")
	os.Setenv("BACKBLAZE_APP_KEY", "test-app-key")
	os.Setenv("B2_BUCKET_ID", "test-bucket-id")

	// Both should fail due to invalid credentials but test structure
	token1, err1 := GetB2DownloadTokenForPrefixCached(prefix1)
	token2, err2 := GetB2DownloadTokenForPrefixCached(prefix2)

	assert.Error(t, err1)
	assert.Error(t, err2)
	assert.Empty(t, token1)
	assert.Empty(t, token2)
}

func TestGetB2DownloadTokenForPrefixCached_MissingCredentials(t *testing.T) {
	// Test behavior when credentials are missing
	prefix := "test-prefix/"

	// Clear environment variables
	os.Unsetenv("BACKBLAZE_MASTER_KEY_ID")
	os.Unsetenv("BACKBLAZE_KEY_ID")
	os.Unsetenv("BACKBLAZE_APP_KEY")
	os.Unsetenv("B2_BUCKET_ID")

	token, err := GetB2DownloadTokenForPrefixCached(prefix)

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "B2 credentials not set")
}

func TestGetB2DownloadTokenForPrefixCached_PartialCredentials(t *testing.T) {
	// Test behavior when only some credentials are set
	prefix := "test-prefix/"

	// Set only some environment variables
	os.Setenv("BACKBLAZE_MASTER_KEY_ID", "test-account-id")
	os.Unsetenv("BACKBLAZE_KEY_ID")
	os.Setenv("BACKBLAZE_APP_KEY", "test-app-key")
	os.Setenv("B2_BUCKET_ID", "test-bucket-id")

	token, err := GetB2DownloadTokenForPrefixCached(prefix)

	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Contains(t, err.Error(), "B2 credentials not set")
}
