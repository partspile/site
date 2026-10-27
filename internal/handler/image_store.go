package handler

import (
	"fmt"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/ui"
)

var adImageStore imagestore.Store
var adImageURLCache *imagestore.URLCache

// proxyAdImages serves img src from this process in local HTTP. Chainguard
// MinIO sends HSTS on plaintext, so browsers HTTPS-upgrade :9000 and fail.
var proxyAdImages bool

// SetAdImageStore replaces the image store (required at startup; for tests use local store).
func SetAdImageStore(store imagestore.Store) {
	adImageStore = store
	urlCache, err := imagestore.NewURLCache(store)
	if err != nil {
		panic("image URL cache: " + err.Error())
	}
	adImageURLCache = urlCache
	_, isMinio := store.(*imagestore.MinioStore)
	proxyAdImages = isMinio && !config.CookieSecure
	ui.SetAdImageURLFunc(resolveAdImageURL)
	ui.SetUserAccountPictureURLFunc(resolveUserAccountPictureURL)
}

func resolveAdImageURL(adID, index int, size string) string {
	if adImageStore == nil || adImageURLCache == nil {
		return ""
	}
	ok, err := adImageStore.Stat(adID, index, size)
	if err != nil {
		logger.Error("ad image stat failed",
			"adID", adID, "index", index, "size", size, "error", err)
	} else if !ok {
		return ""
	}
	if proxyAdImages {
		return fmt.Sprintf("/media/ad/%d/%d/%s", adID, index, size)
	}
	url, err := adImageURLCache.ReusedGetURL(adID, index, size,
		config.MinIOPresignedGetExpiry)
	if err != nil {
		return ""
	}
	return url
}

func resolveUserAccountPictureURL(userID int) string {
	if adImageStore == nil || adImageURLCache == nil {
		return ""
	}
	ok, err := adImageStore.StatUserAccount(userID)
	if err != nil {
		logger.Error("account picture stat failed",
			"userID", userID, "error", err)
	} else if !ok {
		return ""
	}
	url, err := adImageURLCache.ReusedGetUserAccountURL(userID,
		config.MinIOPresignedGetExpiry)
	if err != nil {
		return ""
	}
	return url
}

func deleteUserAccountPicture(userID int) {
	if adImageStore == nil {
		return
	}
	_ = adImageStore.DeleteUserAccount(userID)
	if adImageURLCache != nil {
		adImageURLCache.InvalidateUserAccountURL(userID)
	}
}
