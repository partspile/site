package config

import "time"

const (
	// RedirectDelay is the time to wait before redirecting the user after a successful action.
	RedirectDelay = 1 * time.Second
	// B2TokenCacheDuration is how long to cache B2 download tokens before refreshing.
	B2TokenCacheDuration = 55 * time.Minute
	// B2TokenCacheCleanup is how often the cache cleanup runs for expired tokens.
	B2TokenCacheCleanup = 10 * time.Minute
	// B2DownloadTokenExpiry is the validity duration (in seconds) for B2 download tokens requested from Backblaze.
	B2DownloadTokenExpiry = 3600 // seconds (1 hour)
)
