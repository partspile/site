package config

import "time"

const (
	// RedirectDelay is the time to wait before redirecting the user after a successful action.
	RedirectDelay = 2 * time.Second
)
