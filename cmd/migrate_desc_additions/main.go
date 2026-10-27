package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/logger"
)

func main() {
	dryRun := flag.Bool("dry-run", false,
		"List ads that would change without writing")
	flag.Parse()

	if err := logger.Init("info", "text", ""); err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}
	if err := db.Init(config.DatabaseURL); err != nil {
		logger.Fatal("database", "error", err)
	}
	if err := db.CheckSchema(); err != nil {
		logger.Fatal("schema", "error", err)
	}

	var rows []struct {
		ID          int    `db:"id"`
		Description string `db:"description"`
	}
	if err := db.Select(&rows,
		`SELECT id, description FROM ads ORDER BY id`); err != nil {
		logger.Fatal("select ads", "error", err)
	}

	type update struct {
		id   int
		desc string
	}
	var updates []update
	for _, row := range rows {
		folded, changed := ad.FoldDescriptionAdditions(row.Description)
		if !changed {
			continue
		}
		logger.Info("fold", "adID", row.ID)
		updates = append(updates, update{id: row.ID, desc: folded})
	}

	if *dryRun {
		logger.Info("dry-run complete", "wouldUpdate", len(updates),
			"scanned", len(rows))
		return
	}
	if len(updates) == 0 {
		logger.Info("no description additions to fold", "scanned", len(rows))
		return
	}

	tx, err := db.Begin()
	if err != nil {
		logger.Fatal("begin", "error", err)
	}
	defer tx.Rollback()
	for _, u := range updates {
		if _, err := tx.Exec(
			`UPDATE ads SET description = $1 WHERE id = $2`,
			u.desc, u.id,
		); err != nil {
			logger.Fatal("update", "adID", u.id, "error", err)
		}
	}
	if err := tx.Commit(); err != nil {
		logger.Fatal("commit", "error", err)
	}
	logger.Info("folded description additions",
		"updated", len(updates), "scanned", len(rows))
}
