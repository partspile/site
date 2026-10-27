package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/logger"
)

var imageFilePattern = regexp.MustCompile(`^(\d+)-(\d+w)\.jpg$`)

func main() {
	sourceDir := flag.String("dir", "static/images/ad", "Local directory containing ad images to upload")
	dryRun := flag.Bool("dry-run", false, "List files without uploading")
	flag.Parse()

	if err := logger.Init("info", "text", ""); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	store, err := imagestore.NewDefault()
	if err != nil {
		logger.Fatal("Failed to initialize image store", "error", err)
	}

	info, err := os.Stat(*sourceDir)
	if err != nil {
		logger.Fatal("Source directory not found", "dir", *sourceDir, "error", err)
	}
	if !info.IsDir() {
		logger.Fatal("Source path is not a directory", "dir", *sourceDir)
	}

	var uploaded, skipped, failed int

	err = filepath.WalkDir(*sourceDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(*sourceDir, path)
		if err != nil {
			return err
		}

		adID, index, suffix, ok := parseImageRelPath(filepath.ToSlash(rel))
		if !ok {
			logger.Warn("Skipping unrecognized file", "path", rel)
			skipped++
			return nil
		}

		if *dryRun {
			logger.Info("Would upload", "ad_id", adID, "index", index, "size", suffix, "path", rel)
			uploaded++
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			logger.Error("Failed to read file", "path", path, "error", err)
			failed++
			return nil
		}

		if err := store.Put(adID, index, suffix, data); err != nil {
			logger.Error("Failed to upload", "path", rel, "error", err)
			failed++
			return nil
		}

		logger.Info("Uploaded", "ad_id", adID, "index", index, "size", suffix)
		uploaded++
		return nil
	})
	if err != nil {
		logger.Fatal("Failed to walk source directory", "error", err)
	}

	logger.Info("Migration complete",
		"uploaded", uploaded, "skipped", skipped, "failed", failed, "bucket", config.MinIOBucketName)
	if failed > 0 {
		os.Exit(1)
	}
}

func parseImageRelPath(rel string) (adID, index int, suffix string, ok bool) {
	parts := strings.Split(rel, "/")
	if len(parts) != 2 {
		return 0, 0, "", false
	}

	adID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, "", false
	}

	matches := imageFilePattern.FindStringSubmatch(parts[1])
	if len(matches) != 3 {
		return 0, 0, "", false
	}

	index, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, "", false
	}

	return adID, index, matches[2], true
}
