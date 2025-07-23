package b2util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/parts-pile/site/config"
	"github.com/patrickmn/go-cache"
)

var tokenCache = cache.New(config.B2TokenCacheDuration, config.B2TokenCacheCleanup)

// GetB2DownloadTokenForPrefixCached returns a cached B2 download authorization token for a given ad directory prefix (e.g., "22/")
func GetB2DownloadTokenForPrefixCached(prefix string) (string, error) {
	if token, found := tokenCache.Get(prefix); found {
		return token.(string), nil
	}
	token, err := getB2DownloadTokenForPrefix(prefix)
	if err != nil {
		return "", err
	}
	tokenCache.Set(prefix, token, cache.DefaultExpiration)
	return token, nil
}

// getB2DownloadTokenForPrefix returns a B2 download authorization token for a given ad directory prefix (e.g., "22/")
func getB2DownloadTokenForPrefix(prefix string) (string, error) {
	accountID := config.BackblazeMasterKeyID
	keyID := config.BackblazeKeyID
	appKey := config.BackblazeAppKey
	bucketID := config.B2BucketID
	if accountID == "" || appKey == "" || keyID == "" || bucketID == "" {
		return "", fmt.Errorf("B2 credentials not set")
	}
	req, _ := http.NewRequest("GET", "https://api.backblazeb2.com/b2api/v2/b2_authorize_account", nil)
	req.SetBasicAuth(keyID, appKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("B2 auth error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("B2 auth failed: %s", resp.Status)
	}
	var authResp struct {
		APIURL    string `json:"apiUrl"`
		AuthToken string `json:"authorizationToken"`
		AccountID string `json:"accountId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return "", fmt.Errorf("B2 auth decode error: %w", err)
	}
	apiURL := authResp.APIURL + "/b2api/v2/b2_get_download_authorization"
	expires := int64(config.B2DownloadTokenExpiry) // 1 hour
	body, _ := json.Marshal(map[string]interface{}{
		"bucketId":               bucketID,
		"fileNamePrefix":         prefix,
		"validDurationInSeconds": expires,
	})
	req2, _ := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	req2.Header.Set("Authorization", authResp.AuthToken)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return "", fmt.Errorf("B2 get_download_authorization error: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		return "", fmt.Errorf("B2 get_download_authorization failed: %s", resp2.Status)
	}
	var tokenResp struct {
		AuthorizationToken string `json:"authorizationToken"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("B2 token decode error: %w", err)
	}
	return tokenResp.AuthorizationToken, nil
}
