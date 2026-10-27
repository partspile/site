package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/partspile/site/internal/businesscard"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

func main() {
	categoryFile := flag.String(
		"categories",
		"",
		"Path to ad-category.json (default: embedded list)",
	)
	host := flag.String("host", config.PublicHost(),
		"Site hostname for QR URLs")
	outDir := flag.String("outdir", "cards", "Output directory")
	imagesRoot := flag.String(
		"images",
		"",
		"Directory containing category SVG icons (default: repo static/images/category)",
	)
	flag.Parse()

	if err := logger.Init("info", "text", ""); err != nil {
		fmt.Fprintf(os.Stderr, "logger init: %v\n", err)
		os.Exit(1)
	}

	cats, err := businesscard.LoadCategories(*categoryFile)
	if err != nil {
		logger.Fatal("load categories", "error", err)
	}

	iconsDir := *imagesRoot
	if iconsDir == "" {
		iconsDir = businesscard.ResolveRepoPath("static/images/category")
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		logger.Fatal("create output dir", "error", err)
	}

	for _, cat := range cats {
		path := filepath.Join(*outDir, slugFilename(cat)+".png")
		if err := writeCard(cat, *host, iconsDir, path); err != nil {
			logger.Fatal("generate card", "category", cat.Name, "error", err)
		}
		logger.Info("wrote card", "path", path, "category", cat.Name)
	}
}

func writeCard(
	cat businesscard.Category,
	host, imagesRoot, outPath string,
) error {
	img, err := businesscard.RenderPrintCard(businesscard.PrintCardOptions{
		Category:   cat,
		Host:       host,
		ImagesRoot: imagesRoot,
	})
	if err != nil {
		return err
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return businesscard.WritePNG(f, img)
}

func slugFilename(cat businesscard.Category) string {
	slug := strings.ToLower(cat.Name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, "&", "and")
	return "partspile-" + slug
}
