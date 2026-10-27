package imagestore

import (
	"fmt"
	"time"

	"github.com/partspile/site/internal/cache"
)

// ImageSizes are the JPEG derivative suffixes stored per ad image.
var ImageSizes = []string{"160w", "480w", "1200w"}

// ImageRef identifies one stored ad image variant.
type ImageRef struct {
	Index  int
	Suffix string
}

// Store persists ad image files in MinIO. LocalStore exists for tests only.
type Store interface {
	Put(adID, index int, suffix string, data []byte) error
	Get(adID, index int, suffix string) ([]byte, error)
	Stat(adID, index int, suffix string) (bool, error)
	ListAd(adID int) ([]ImageRef, error)
	DeleteAd(adID int) error
	PresignPut(adID, index int, suffix string,
		expiry time.Duration) (string, error)
	PresignGet(adID, index int, suffix string,
		expiry time.Duration) (string, error)

	PutUserAccount(userID int, data []byte) error
	GetUserAccount(userID int) ([]byte, error)
	StatUserAccount(userID int) (bool, error)
	DeleteUserAccount(userID int) error
	PresignPutUserAccount(userID int, expiry time.Duration) (string, error)
	PresignGetUserAccount(userID int, expiry time.Duration) (string, error)
}

// URLCache reuses PresignGet URLs for the signature lifetime (TTL = expiry).
type URLCache struct {
	store Store
	urls  *cache.Cache[string]
}

func NewURLCache(store Store) (*URLCache, error) {
	urls, err := cache.New(func(string) int64 { return 1 },
		"Presigned GET URLs")
	if err != nil {
		return nil, err
	}
	return &URLCache{store: store, urls: urls}, nil
}

func cacheKey(adID, index int, suffix string) string {
	return fmt.Sprintf("%d/%d/%s", adID, index, suffix)
}

func userAccountCacheKey(userID int) string {
	return fmt.Sprintf("users/%d/account", userID)
}

// ReusedGetURL returns a stable PresignGet URL. Miss (or TTL expiry) mints fresh.
func (c *URLCache) ReusedGetURL(adID, index int, suffix string,
	expiry time.Duration) (string, error) {
	key := cacheKey(adID, index, suffix)
	if url, ok := c.urls.Get(key); ok {
		return url, nil
	}

	url, err := c.store.PresignGet(adID, index, suffix, expiry)
	if err != nil {
		return "", err
	}
	c.urls.SetWithTTL(key, url, 1, expiry)
	return url, nil
}

// InvalidateGetURL drops a cached GET URL.
func (c *URLCache) InvalidateGetURL(adID, index int, suffix string) {
	c.urls.Del(cacheKey(adID, index, suffix))
}

// ReusedGetUserAccountURL returns a stable PresignGet for a user account picture.
func (c *URLCache) ReusedGetUserAccountURL(userID int,
	expiry time.Duration) (string, error) {
	key := userAccountCacheKey(userID)
	if url, ok := c.urls.Get(key); ok {
		return url, nil
	}

	url, err := c.store.PresignGetUserAccount(userID, expiry)
	if err != nil {
		return "", err
	}
	c.urls.SetWithTTL(key, url, 1, expiry)
	return url, nil
}

// InvalidateUserAccountURL drops a cached user account picture GET URL.
func (c *URLCache) InvalidateUserAccountURL(userID int) {
	c.urls.Del(userAccountCacheKey(userID))
}
