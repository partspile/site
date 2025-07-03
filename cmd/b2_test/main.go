package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"gopkg.in/kothar/go-backblaze.v0"
)

func main() {
	accountID := os.Getenv("BACKBLAZE_KEY_ID")
	appKey := os.Getenv("BACKBLAZE_APP_KEY")
	fmt.Println(accountID, appKey, len(accountID), len(appKey))
	if accountID == "" || appKey == "" {
		log.Fatal("B2 credentials not set in env vars")
	}
	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      accountID,
		ApplicationKey: appKey,
	})
	if err != nil {
		log.Fatal("B2 auth error:", err)
	}
	bucket, err := b2.Bucket("parts-pile")
	if err != nil {
		log.Fatal("B2 bucket error:", err)
	}
	// Dummy .webp bytes (not a real image, just for test)
	dummy := []byte("RIFFxxxxWEBPVP8 ")
	name := "b2_test_upload.webp"
	_, err = bucket.UploadTypedFile(name, "image/webp", nil, bytes.NewReader(dummy))
	if err != nil {
		log.Fatal("B2 upload error:", err)
	}
	fmt.Println("B2 test upload succeeded:", name)
}
