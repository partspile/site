package minio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/parts-pile/site/config"
)

var minioClient *minio.Client

// Init initializes the MinIO client. This should be called during application startup.
// If initialization fails, the application should exit.
func Init() error {
	// Initialize MinIO client
	if config.MinIOAPIURL == "" {
		return fmt.Errorf("MINIO_API_URL environment variable not set")
	}
	if config.MinIOUsername == "" {
		return fmt.Errorf("MINIO_USERNAME environment variable not set")
	}
	if config.MinIOPassword == "" {
		return fmt.Errorf("MINIO_PASSWORD environment variable not set")
	}

	// Parse the API URL to extract endpoint
	// If URL doesn't have a scheme, add http:// as default
	apiURL := config.MinIOAPIURL
	if !strings.Contains(apiURL, "://") {
		apiURL = "http://" + apiURL
	}

	endpointURL, err := url.Parse(apiURL)
	if err != nil {
		return fmt.Errorf("invalid MinIO API URL %q: %w", config.MinIOAPIURL, err)
	}

	// Validate that we have a host
	endpoint := endpointURL.Host
	if endpoint == "" {
		// If Host is still empty after parsing, the URL format is invalid
		return fmt.Errorf("MinIO API URL missing host: %q (expected format: http://host:port or https://host:port, got: %q)", config.MinIOAPIURL, apiURL)
	}

	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.MinIOUsername, config.MinIOPassword, ""),
		Secure: endpointURL.Scheme == "https",
	})
	if err != nil {
		return fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	// Ensure bucket exists
	ctx := context.Background()
	bucketExists, err := minioClient.BucketExists(ctx, config.MinIOBucketName)
	if err != nil {
		return fmt.Errorf("failed to check if bucket exists: %w", err)
	}

	if !bucketExists {
		err = minioClient.MakeBucket(ctx, config.MinIOBucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket %q: %w", config.MinIOBucketName, err)
		}
	}

	return nil
}

// GetImageURL returns the app proxy URL for a given ad and image index
func GetImageURL(adID, imageIndex int, size string) (string, error) {
	return fmt.Sprintf("/image/%d/%d/%s", adID, imageIndex, size), nil
}

// PutImageObject uploads an image object to MinIO
func PutImageObject(adID, imageIndex int, size string, data []byte) error {
	imagePath := filepath.Join(
		fmt.Sprintf("%d", adID),
		fmt.Sprintf("%d-%s.webp", imageIndex, size),
	)

	ctx := context.Background()
	_, err := minioClient.PutObject(ctx, config.MinIOBucketName, imagePath,
		bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: "image/webp"})
	if err != nil {
		return fmt.Errorf("failed to put object to MinIO: %w", err)
	}

	return nil
}

// GetImageObject retrieves an image object from MinIO and returns its data
func GetImageObject(adID, imageIndex int, size string) ([]byte, error) {
	imagePath := filepath.Join(
		fmt.Sprintf("%d", adID),
		fmt.Sprintf("%d-%s.webp", imageIndex, size),
	)

	ctx := context.Background()
	obj, err := minioClient.GetObject(ctx, config.MinIOBucketName, imagePath, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object from MinIO: %w", err)
	}
	defer obj.Close()

	// Read the object data
	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	return data, nil
}

// ListAdImages lists all image objects for a given ad in MinIO
func ListAdImages(adID int) ([]string, error) {
	prefix := fmt.Sprintf("%d/", adID)
	ctx := context.Background()

	var imagePaths []string
	objectCh := minioClient.ListObjects(ctx, config.MinIOBucketName,
		minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		})

	for obj := range objectCh {
		if obj.Err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", obj.Err)
		}
		imagePaths = append(imagePaths, obj.Key)
	}

	return imagePaths, nil
}

// DeleteImageObject deletes an image object from MinIO
func DeleteImageObject(adID, imageIndex int, size string) error {
	imagePath := filepath.Join(
		fmt.Sprintf("%d", adID),
		fmt.Sprintf("%d-%s.webp", imageIndex, size),
	)

	ctx := context.Background()
	err := minioClient.RemoveObject(ctx, config.MinIOBucketName, imagePath,
		minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete object from MinIO: %w", err)
	}

	return nil
}

// DeleteAdImages deletes all images for a given ad from MinIO
func DeleteAdImages(adID int) error {
	imagePaths, err := ListAdImages(adID)
	if err != nil {
		return fmt.Errorf("failed to list images for ad %d: %w", adID, err)
	}

	ctx := context.Background()
	errorCh := minioClient.RemoveObjects(ctx, config.MinIOBucketName,
		func() <-chan minio.ObjectInfo {
			objectCh := make(chan minio.ObjectInfo, 1)
			go func() {
				defer close(objectCh)
				for _, path := range imagePaths {
					objectCh <- minio.ObjectInfo{Key: path}
				}
			}()
			return objectCh
		}(),
		minio.RemoveObjectsOptions{})

	var deleteErrors []error
	for err := range errorCh {
		deleteErrors = append(deleteErrors, err.Err)
	}

	if len(deleteErrors) > 0 {
		return fmt.Errorf("failed to delete %d images: %v",
			len(deleteErrors), deleteErrors[0])
	}

	return nil
}
