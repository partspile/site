package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/imgconv"
	"github.com/partspile/site/internal/logger"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "List conversions without writing")
	quality := flag.Int("quality", imgconv.DefaultQuality,
		"JPEG quality 1-100")
	flag.Parse()

	if err := logger.Init("info", "text", ""); err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		os.Exit(1)
	}

	store, err := imagestore.NewMinio()
	if err != nil {
		logger.Fatal("minio", "error", err)
	}

	keys, err := store.ListKeys()
	if err != nil {
		logger.Fatal("list", "error", err)
	}

	var converted, skipped, failed int
	for _, key := range keys {
		dest, ok := jpegDestKey(key)
		if !ok {
			logger.Info("skip unrecognized", "key", key)
			skipped++
			continue
		}

		data, err := store.ReadKey(key)
		if err != nil {
			logger.Error("read", "key", key, "error", err)
			failed++
			continue
		}

		if dest == key && isJPEG(data) {
			skipped++
			continue
		}

		exists, err := store.KeyExists(dest)
		if err != nil {
			logger.Error("stat dest", "key", dest, "error", err)
			failed++
			continue
		}
		if dest != key && exists {
			destData, err := store.ReadKey(dest)
			if err != nil {
				logger.Error("read dest", "key", dest, "error", err)
				failed++
				continue
			}
			if isJPEG(destData) {
				if *dryRun {
					logger.Info("would delete leftover", "key", key)
				} else if err := store.DeleteKey(key); err != nil {
					logger.Error("delete leftover", "key", key,
						"error", err)
					failed++
					continue
				}
				converted++
				continue
			}
		}

		jpeg, err := imgconv.ToJPEG(data, *quality)
		if err != nil {
			logger.Error("convert", "key", key, "error", err)
			failed++
			continue
		}

		if *dryRun {
			logger.Info("would convert", "from", key, "to", dest,
				"src_bytes", len(data), "jpg_bytes", len(jpeg))
			converted++
			continue
		}

		if err := store.WriteJPEG(dest, jpeg); err != nil {
			logger.Error("write", "key", dest, "error", err)
			failed++
			continue
		}
		if dest != key {
			if err := store.DeleteKey(key); err != nil {
				logger.Error("delete source", "key", key, "error", err)
				failed++
				continue
			}
		}
		logger.Info("converted", "from", key, "to", dest,
			"src_bytes", len(data), "jpg_bytes", len(jpeg))
		converted++
	}

	logger.Info("done",
		"converted", converted, "skipped", skipped, "failed", failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func isJPEG(data []byte) bool {
	return len(data) >= 3 &&
		data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff
}

func jpegDestKey(key string) (string, bool) {
	ext := strings.ToLower(path.Ext(key))
	switch ext {
	case ".jpg", ".jpeg", ".webp", ".png":
	default:
		return "", false
	}
	base := strings.TrimSuffix(key, path.Ext(key))
	return base + ".jpg", true
}
