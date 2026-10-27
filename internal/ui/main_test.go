package ui

import (
	"os"
	"testing"

	"github.com/partspile/site/internal/config"
)

func TestMain(m *testing.M) {
	config.ServerName = "Parts Pile"
	config.ContactEmail = "contact@parts-pile.com"
	os.Exit(m.Run())
}
