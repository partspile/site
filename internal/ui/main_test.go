package ui

import (
	"os"
	"testing"

	"github.com/rocky-ads/site/internal/config"
)

func TestMain(m *testing.M) {
	config.ServerName = "Rocky Ads"
	config.ContactEmail = "contact@rockyads.com"
	os.Exit(m.Run())
}
