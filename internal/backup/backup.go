package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/encryption"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/journal"
	"github.com/partspile/site/internal/logger"
)

func runBackup(outDir string, store imagestore.Store, dryRun, verbose bool) error {
	var adRows []adDBRow
	err := db.Select(&adRows, `
		SELECT id, category_id, title, description, created_at, expires_at,
		       inactive_at, deleted_at, user_id, image_count, location_id, tags,
		       CASE WHEN embedding IS NOT NULL
		            THEN embedding::text END AS embedding_text,
		       vector_metadata::text AS vector_metadata
		FROM ads
		ORDER BY created_at, id`)
	if err != nil {
		return fmt.Errorf("query ads: %w", err)
	}
	if len(adRows) == 0 {
		logger.Info("No ads to backup")
	}

	adIDToRef := make(map[int]int, len(adRows))
	adRefs := make([]int, len(adRows))
	for i, a := range adRows {
		adIDToRef[a.ID] = i
		adRefs[i] = a.ID
	}

	locationIDs := make([]int, 0, len(adRows))
	for _, a := range adRows {
		if a.LocationID != nil {
			locationIDs = append(locationIDs, *a.LocationID)
		}
	}

	var bookmarkDB []bookmarkDBRow
	var clickDB []userAdClickDBRow
	var imageClickDB []userAdImageClickDBRow
	var facetDB []adFacetDBRow
	var convDB []conversationDBRow
	var opinionDB []rockOpinionDBRow

	if len(adRefs) > 0 {
		clause, args := intInClause(adRefs)
		if err := db.Select(&bookmarkDB, fmt.Sprintf(`
			SELECT user_id, ad_id, bookmarked_at
			FROM bookmarks WHERE ad_id IN (%s)`, clause), args...); err != nil {
			return fmt.Errorf("query bookmarks: %w", err)
		}
		if err := db.Select(&clickDB, fmt.Sprintf(`
			SELECT ad_id, user_id, click_count, last_clicked_at
			FROM user_ad_clicks WHERE ad_id IN (%s)`, clause), args...); err != nil {
			return fmt.Errorf("query user_ad_clicks: %w", err)
		}
		if err := db.Select(&imageClickDB, fmt.Sprintf(`
			SELECT ad_id, user_id, image_index, click_count, last_clicked_at
			FROM user_ad_image_clicks WHERE ad_id IN (%s)`, clause), args...); err != nil {
			return fmt.Errorf("query user_ad_image_clicks: %w", err)
		}
		if err := db.Select(&facetDB, fmt.Sprintf(`
			SELECT ad_id, "key", num, text
			FROM ad_facets WHERE ad_id IN (%s)`, clause), args...); err != nil {
			return fmt.Errorf("query ad_facets: %w", err)
		}
		if err := db.Select(&convDB, fmt.Sprintf(`
			SELECT id, ad_id, owner_id, inquirer_id,
			       owner_has_unread, inquirer_has_unread,
			       rock_thrower_id, rock_thrown_at, journal
			FROM conversations WHERE ad_id IN (%s)
			ORDER BY id`, clause), args...); err != nil {
			return fmt.Errorf("query conversations: %w", err)
		}
	}

	convIDToRef := make(map[int]int, len(convDB))
	for i, c := range convDB {
		convIDToRef[c.ID] = i
	}
	locationIDs = uniqueInts(locationIDs)

	var msgDB []messageDBRow
	for _, c := range convDB {
		plainJournal, err := encryption.Open(
			c.ID, c.Journal, config.DBEncryptionKey,
		)
		if err != nil {
			return fmt.Errorf("open journal for conversation %d: %w", c.ID, err)
		}
		for _, e := range journal.Parse(plainJournal) {
			if e.Kind != journal.Message {
				continue
			}
			msgDB = append(msgDB, messageDBRow{
				ConversationID: c.ID,
				SenderID:       e.UserID,
				Content:        e.Body,
				CreatedAt:      e.At,
			})
		}
	}

	if len(convIDToRef) > 0 {
		convIDs := make([]int, 0, len(convIDToRef))
		for id := range convIDToRef {
			convIDs = append(convIDs, id)
		}
		clause, args := intInClause(convIDs)
		if err := db.Select(&opinionDB, fmt.Sprintf(`
			SELECT conversation_id, generated_at, summary, assessment,
			       assessment_detail, resolution, reasoning
			FROM rock_opinions WHERE conversation_id IN (%s)`, clause), args...); err != nil {
			return fmt.Errorf("query rock_opinions: %w", err)
		}
	}

	// All users — including admins with no ads/activity.
	var userDB []userDBRow
	if err := db.Select(&userDB, `
		SELECT id, created_at, deleted_at, is_admin,
		       encrypted_name, name_nonce, name_hash,
		       password_hash, password_salt, password_algo,
		       encrypted_phone, phone_nonce, phone_hash,
		       phone_verified, sms_opted_out, last_sms_sent_at,
		       has_account_picture, account_picture_url
		FROM users
		ORDER BY id`); err != nil {
		return fmt.Errorf("query users: %w", err)
	}

	userIDToHash := make(map[int]string, len(userDB))
	users := make([]UserRow, len(userDB))
	for i, u := range userDB {
		userIDToHash[u.ID] = u.NameHash
		users[i] = userRowFromDB(u)
		if err := verifyUserDecrypt(users[i]); err != nil {
			return err
		}
	}

	locationIDToRaw := make(map[int]string)
	var locations []LocationRow
	if len(locationIDs) > 0 {
		clause, args := intInClause(locationIDs)
		if err := db.Select(&locations, fmt.Sprintf(`
			SELECT raw_text, city, admin_area, country, latitude, longitude
			FROM locations WHERE id IN (%s)`, clause), args...); err != nil {
			return fmt.Errorf("query locations: %w", err)
		}
		var locDB []struct {
			ID      int    `db:"id"`
			RawText string `db:"raw_text"`
		}
		if err := db.Select(&locDB, fmt.Sprintf(`
			SELECT id, raw_text FROM locations WHERE id IN (%s)`,
			clause), args...); err != nil {
			return fmt.Errorf("query location ids: %w", err)
		}
		for _, l := range locDB {
			locationIDToRaw[l.ID] = l.RawText
		}
	}

	ads := make([]AdRow, len(adRows))
	embeddingCount := 0
	for i, a := range adRows {
		row := AdRow{
			Ref:         i,
			CategoryID:  a.CategoryID,
			Title:       a.Title,
			Description: a.Description,
			CreatedAt:   a.CreatedAt,
			ExpiresAt:   a.ExpiresAt,
			InactiveAt:  a.InactiveAt,
			DeletedAt:   a.DeletedAt,
			OwnerHash:   userIDToHash[a.UserID],
			ImageCount:  a.ImageCount,
			Tags:        a.Tags,
		}
		if a.LocationID != nil {
			if raw, ok := locationIDToRaw[*a.LocationID]; ok {
				row.LocationRaw = &raw
			}
		}
		if a.EmbeddingText != nil && *a.EmbeddingText != "" {
			emb, err := encodeEmbedding(*a.EmbeddingText)
			if err != nil {
				return fmt.Errorf("encode embedding ad %d: %w", a.ID, err)
			}
			row.Embedding = emb
			embeddingCount++
		}
		if a.VectorMetadata != nil && *a.VectorMetadata != "" {
			row.VectorMetadata = json.RawMessage(*a.VectorMetadata)
		}
		ads[i] = row
	}

	facets := make([]AdFacetRow, len(facetDB))
	for i, f := range facetDB {
		facets[i] = AdFacetRow{
			AdRef: adIDToRef[f.AdID],
			Key:   f.Key,
			Num:   f.Num,
			Text:  f.Text,
		}
	}

	bookmarks := make([]BookmarkRow, len(bookmarkDB))
	for i, b := range bookmarkDB {
		bookmarks[i] = BookmarkRow{
			UserHash:     userIDToHash[b.UserID],
			AdRef:        adIDToRef[b.AdID],
			BookmarkedAt: b.BookmarkedAt,
		}
	}

	clicks := make([]UserAdClickRow, len(clickDB))
	for i, c := range clickDB {
		clicks[i] = UserAdClickRow{
			AdRef:         adIDToRef[c.AdID],
			UserHash:      userIDToHash[c.UserID],
			ClickCount:    c.ClickCount,
			LastClickedAt: c.LastClickedAt,
		}
	}

	imageClicks := make([]UserAdImageClickRow, len(imageClickDB))
	for i, c := range imageClickDB {
		imageClicks[i] = UserAdImageClickRow{
			AdRef:         adIDToRef[c.AdID],
			UserHash:      userIDToHash[c.UserID],
			ImageIndex:    c.ImageIndex,
			ClickCount:    c.ClickCount,
			LastClickedAt: c.LastClickedAt,
		}
	}

	conversations := make([]ConversationRow, len(convDB))
	for i, c := range convDB {
		row := ConversationRow{
			Ref:               i,
			AdRef:             adIDToRef[c.AdID],
			OwnerHash:         userIDToHash[c.OwnerID],
			InquirerHash:      userIDToHash[c.InquirerID],
			OwnerHasUnread:    c.OwnerHasUnread,
			InquirerHasUnread: c.InquirerHasUnread,
			RockThrownAt:      c.RockThrownAt,
		}
		if c.RockThrowerID != nil {
			h := userIDToHash[*c.RockThrowerID]
			row.RockThrowerHash = &h
		}
		conversations[i] = row
	}

	messages := make([]MessageRow, len(msgDB))
	for i, m := range msgDB {
		messages[i] = MessageRow{
			ConversationRef: convIDToRef[m.ConversationID],
			SenderHash:      userIDToHash[m.SenderID],
			Content:         m.Content,
			CreatedAt:       m.CreatedAt,
		}
	}

	opinions := make([]RockOpinionRow, len(opinionDB))
	for i, o := range opinionDB {
		opinions[i] = RockOpinionRow{
			ConversationRef:  convIDToRef[o.ConversationID],
			GeneratedAt:      o.GeneratedAt,
			Summary:          o.Summary,
			Assessment:       o.Assessment,
			AssessmentDetail: o.AssessmentDetail,
			Resolution:       o.Resolution,
			Reasoning:        o.Reasoning,
		}
	}

	imageCount := 0
	if dryRun {
		for ref, a := range adRows {
			if a.ImageCount == 0 {
				continue
			}
			refs, err := store.ListAd(a.ID)
			if err != nil {
				return fmt.Errorf("list images for ad %d: %w", a.ID, err)
			}
			imageCount += len(refs)
			if verbose {
				logger.Info("Would backup image", "ad_ref", ref, "count", len(refs))
			}
		}
		accountCount, err := backupUserAccounts(
			"", store, userDB, true, verbose)
		if err != nil {
			return err
		}
		logger.Info("Dry run complete",
			"ads", len(ads), "users", len(users),
			"conversations", len(conversations), "images", imageCount,
			"account_pictures", accountCount)
		return nil
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("create backup dir: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(outDir, dirImages), 0755); err != nil {
		return fmt.Errorf("create images dir: %w", err)
	}

	for ref, a := range adRows {
		if a.ImageCount == 0 {
			continue
		}
		refs, err := store.ListAd(a.ID)
		if err != nil {
			return fmt.Errorf("list images for ad %d: %w", a.ID, err)
		}
		adDir := filepath.Join(outDir, dirImages, fmt.Sprintf("%d", ref))
		if err := os.MkdirAll(adDir, 0755); err != nil {
			return fmt.Errorf("create ad image dir: %w", err)
		}
		for _, img := range refs {
			data, err := store.Get(a.ID, img.Index, img.Suffix)
			if err != nil {
				return fmt.Errorf("get image ad %d index %d: %w",
					a.ID, img.Index, err)
			}
			name := fmt.Sprintf("%d-%s.jpg", img.Index, img.Suffix)
			path := filepath.Join(adDir, name)
			if err := os.WriteFile(path, data, 0644); err != nil {
				return fmt.Errorf("write image %s: %w", path, err)
			}
			imageCount++
			if verbose {
				logger.Info("Backed up image", "ad_ref", ref, "path", path)
			}
		}
	}

	accountCount, err := backupUserAccounts(
		outDir, store, userDB, false, verbose)
	if err != nil {
		return err
	}

	manifest := Manifest{
		CreatedAt: time.Now().UTC(),
		Counts: Counts{
			Users:             len(users),
			Locations:         len(locations),
			Ads:               len(ads),
			AdFacets:          len(facets),
			Bookmarks:         len(bookmarks),
			UserAdClicks:      len(clicks),
			UserAdImageClicks: len(imageClicks),
			Conversations:     len(conversations),
			Messages:          len(messages),
			RockOpinions:      len(opinions),
			Images:            imageCount,
			AccountPictures:   accountCount,
			Embeddings:        embeddingCount,
		},
	}

	files := []struct {
		name string
		data any
	}{
		{fileManifest, manifest},
		{fileUsers, users},
		{fileLocations, locations},
		{fileAds, ads},
		{fileAdFacets, facets},
		{fileBookmarks, bookmarks},
		{fileUserAdClicks, clicks},
		{fileUserAdImageClicks, imageClicks},
		{fileConversations, conversations},
		{fileMessages, messages},
		{fileRockOpinions, opinions},
	}
	for _, f := range files {
		if err := writeJSON(filepath.Join(outDir, f.name), f.data); err != nil {
			return err
		}
	}

	logger.Info("Backup staged",
		"dir", outDir, "ads", len(ads), "images", imageCount,
		"account_pictures", accountCount)
	return nil
}

func backupUserAccounts(outDir string, store imagestore.Store,
	users []userDBRow, dryRun, verbose bool) (int, error) {
	count := 0
	for _, u := range users {
		if u.HasAccountPicture == 0 {
			continue
		}
		ok, err := store.StatUserAccount(u.ID)
		if err != nil {
			return 0, fmt.Errorf(
				"stat account picture user %d: %w", u.ID, err)
		}
		if !ok {
			logger.Warn("Account picture missing", "user_id", u.ID)
			continue
		}
		if dryRun {
			count++
			if verbose {
				logger.Info("Would backup account picture",
					"user_id", u.ID)
			}
			continue
		}
		data, err := store.GetUserAccount(u.ID)
		if err != nil {
			return 0, fmt.Errorf(
				"get account picture user %d: %w", u.ID, err)
		}
		dir := filepath.Join(outDir, dirUserAccounts, u.NameHash)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return 0, fmt.Errorf("create user account dir: %w", err)
		}
		path := filepath.Join(dir, "account.jpg")
		if err := os.WriteFile(path, data, 0644); err != nil {
			return 0, fmt.Errorf("write %s: %w", path, err)
		}
		count++
		if verbose {
			logger.Info("Backed up account picture",
				"user_id", u.ID, "path", path)
		}
	}
	return count, nil
}

func writeJSON(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", path, err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
