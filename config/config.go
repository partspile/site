package config

import "time"

const (
	// SessionDuration is the length of time a user session is valid.
	SessionDuration = 24 * time.Hour

	// SessionCleanupInterval is the frequency at which expired sessions are cleared from memory.
	SessionCleanupInterval = 10 * time.Minute

	// RedirectDelay is the time to wait before redirecting the user after a successful action.
	RedirectDelay = 2 * time.Second
)
