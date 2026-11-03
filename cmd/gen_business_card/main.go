package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/partspile/site/internal/businesscard"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

func main() {
	host := flag.String("host", config.PublicHost(),
		"Site hostname for QR URLs")
	outDir := flag.String("outdir", "cards", "Output directory")
	outName := flag.String("out", "partspile.png", "Output PNG filename")
	imagesRoot := flag.String(
		"images",
		"",
		"Directory containing flag.svg and block.svg (default: repo static/images)",
	)
	flag.Parse()

	if err := logger.Init("info", "text", ""); err != nil {
		fmt.Fprintf(os.Stderr, "logger init: %v\n", err)
		os.Exit(1)
	}

	iconsDir := *imagesRoot
	if iconsDir == "" {
		iconsDir = businesscard.ResolveRepoPath("static/images")
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		logger.Fatal("create output dir", "error", err)
	}

	path := filepath.Join(*outDir, *outName)
	if err := writeCard(*host, iconsDir, path); err != nil {
		logger.Fatal("generate card", "error", err)
	}
	logger.Info("wrote card", "path", path)
}

func writeCard(host, imagesRoot, outPath string) error {
	img, err := businesscard.RenderPrintCard(businesscard.PrintCardOptions{
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
