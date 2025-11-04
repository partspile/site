package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/memory"
	"github.com/parts-pile/site/config"
)

// sessionStore is the session store used for CSRF token management
var sessionStore = session.New(session.Config{
	Storage:        memory.New(),
	KeyLookup:      "cookie:session_id",
	Expiration:     24 * time.Hour,
	CookieSecure:   config.CookieSecure,
	CookieSameSite: "Strict",
	CookieHTTPOnly: true,
})
