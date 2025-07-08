package b2util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/patrickmn/go-cache"
)

var tokenCache = cache.New(55*time.Minute, 10*time.Minute)

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
	accountID := os.Getenv("BACKBLAZE_MASTER_KEY_ID")
	keyID := os.Getenv("BACKBLAZE_KEY_ID")
	appKey := os.Getenv("BACKBLAZE_APP_KEY")
	bucketID := os.Getenv("B2_BUCKET_ID")
	if accountID == "" || appKey == "" || keyID == "" || bucketID == "" {
		return "", fmt.Errorf("B2 credentials not set")
	}
	// Authorize account
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
	// Call b2_get_download_authorization for the prefix
	apiURL := authResp.APIURL + "/b2api/v2/b2_get_download_authorization"
	expires := int64(3600) // 1 hour
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

// ListFilesWithPrefix lists file names in the B2 bucket for a given prefix (e.g., '24/')
func ListFilesWithPrefix(prefix string) ([]string, error) {
	accountID := os.Getenv("BACKBLAZE_MASTER_KEY_ID")
	keyID := os.Getenv("BACKBLAZE_KEY_ID")
	appKey := os.Getenv("BACKBLAZE_APP_KEY")
	bucketID := os.Getenv("B2_BUCKET_ID")
	if accountID == "" || appKey == "" || keyID == "" || bucketID == "" {
		return nil, fmt.Errorf("B2 credentials not set")
	}
	// Authorize account
	req, _ := http.NewRequest("GET", "https://api.backblazeb2.com/b2api/v2/b2_authorize_account", nil)
	req.SetBasicAuth(keyID, appKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("B2 auth error: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("B2 auth failed: %s", resp.Status)
	}
	var authResp struct {
		APIURL    string `json:"apiUrl"`
		AuthToken string `json:"authorizationToken"`
		AccountID string `json:"accountId"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("B2 auth decode error: %w", err)
	}
	// List files with prefix
	apiURL := authResp.APIURL + "/b2api/v2/b2_list_file_names"
	body, _ := json.Marshal(map[string]interface{}{
		"bucketId":     bucketID,
		"prefix":       prefix,
		"maxFileCount": 1000,
	})
	req2, _ := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	req2.Header.Set("Authorization", authResp.AuthToken)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := http.DefaultClient.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("B2 list_file_names error: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		return nil, fmt.Errorf("B2 list_file_names failed: %s", resp2.Status)
	}
	var listResp struct {
		Files []struct {
			FileName string `json:"fileName"`
		} `json:"files"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("B2 list decode error: %w", err)
	}
	fileNames := make([]string, 0, len(listResp.Files))
	for _, f := range listResp.Files {
		fileNames = append(fileNames, f.FileName)
	}
	return fileNames, nil
}
