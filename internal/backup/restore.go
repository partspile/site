package backup

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/dbinit"
	"github.com/partspile/site/internal/encryption"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/imgconv"
	"github.com/partspile/site/internal/journal"
	"github.com/partspile/site/internal/logger"
	"github.com/pgvector/pgvector-go"
)

var restoreImagePattern = regexp.MustCompile(
	`^(\d+)-(\d+w)\.(jpg|jpeg|webp|png)$`)

func runRestore(fromDir string, store imagestore.Store, backupKey []byte,
	dryRun, verbose bool) error {
	var manifest Manifest
	if err := readJSON(filepath.Join(fromDir, fileManifest), &manifest); err != nil {
		return err
	}

	var users []UserRow
	if err := readJSON(filepath.Join(fromDir, fileUsers), &users); err != nil {
		return err
	}
	var locations []LocationRow
	if err := readJSON(filepath.Join(fromDir, fileLocations), &locations); err != nil {
		return err
	}
	var ads []AdRow
	if err := readJSON(filepath.Join(fromDir, fileAds), &ads); err != nil {
		return err
	}
	var facets []AdFacetRow
	if err := readJSON(filepath.Join(fromDir, fileAdFacets), &facets); err != nil {
		return err
	}
	var bookmarks []BookmarkRow
	if err := readJSON(filepath.Join(fromDir, fileBookmarks), &bookmarks); err != nil {
		return err
	}
	var clicks []UserAdClickRow
	if err := readJSON(filepath.Join(fromDir, fileUserAdClicks), &clicks); err != nil {
		return err
	}
	var imageClicks []UserAdImageClickRow
	if err := readJSON(filepath.Join(fromDir, fileUserAdImageClicks), &imageClicks); err != nil {
		return err
	}
	var conversations []ConversationRow
	if err := readJSON(filepath.Join(fromDir, fileConversations), &conversations); err != nil {
		return err
	}
	var messages []MessageRow
	if err := readJSON(filepath.Join(fromDir, fileMessages), &messages); err != nil {
		return err
	}
	var opinions []RockOpinionRow
	if err := readJSON(filepath.Join(fromDir, fileRockOpinions), &opinions); err != nil {
		return err
	}

	if dryRun {
		logger.Info("Dry run restore",
			"users", len(users), "locations", len(locations),
			"ads", len(ads), "conversations", len(conversations),
			"images", manifest.Counts.Images,
			"account_pictures", manifest.Counts.AccountPictures)
		return nil
	}

	if err := prepareFreshDatabase(); err != nil {
		return err
	}

	sort.Slice(ads, func(i, j int) bool {
		return ads[i].Ref < ads[j].Ref
	})

	// Live users before deleted so a recycled phone_hash never collides
	// with the partial unique index on live phone_hash.
	sort.SliceStable(users, func(i, j int) bool {
		iDel := users[i].DeletedAt != nil
		jDel := users[j].DeletedAt != nil
		if iDel != jDel {
			return !iDel && jDel
		}
		return false
	})

	userHashToID := make(map[string]int, len(users))
	for _, u := range users {
		id, err := resolveUserID(u, backupKey)
		if err != nil {
			return err
		}
		userHashToID[u.NameHash] = id
	}

	locationRawToID := make(map[string]int, len(locations))
	for _, loc := range locations {
		id, err := resolveLocationID(loc)
		if err != nil {
			return err
		}
		locationRawToID[loc.RawText] = id
	}

	adRefToID := make(map[int]int, len(ads))
	for _, a := range ads {
		ownerID, ok := userHashToID[a.OwnerHash]
		if !ok {
			return fmt.Errorf("unknown owner hash for ad ref %d", a.Ref)
		}
		var locationID any
		if a.LocationRaw != nil {
			locID, ok := locationRawToID[*a.LocationRaw]
			if !ok {
				return fmt.Errorf(
					"unknown location %q for ad ref %d", *a.LocationRaw, a.Ref,
				)
			}
			locationID = locID
		}
		var emb any
		if len(a.Embedding) > 0 {
			floats, err := decodeEmbedding(a.Embedding)
			if err != nil {
				return fmt.Errorf("decode embedding ad ref %d: %w", a.Ref, err)
			}
			emb = pgvector.NewVector(floats)
		}
		var meta any
		if len(a.VectorMetadata) > 0 {
			meta = string(a.VectorMetadata)
		}
		var newID int
		err := db.QueryRow(`
			INSERT INTO ads (
				category_id, title, description, created_at, expires_at,
				inactive_at, deleted_at, user_id, image_count,
				location_id, tags, embedding, vector_metadata
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12,
			          $13::jsonb)
			RETURNING id`,
			a.CategoryID, a.Title, a.Description, a.CreatedAt, a.ExpiresAt,
			a.InactiveAt, a.DeletedAt, ownerID, a.ImageCount,
			locationID, a.Tags, emb, meta,
		).Scan(&newID)
		if err != nil {
			return fmt.Errorf("insert ad ref %d: %w", a.Ref, err)
		}
		adRefToID[a.Ref] = newID
	}

	if _, err := db.Exec(`
		UPDATE ads a
		SET latitude = l.latitude, longitude = l.longitude
		FROM locations l
		WHERE a.location_id = l.id`); err != nil {
		return fmt.Errorf("backfill ad coordinates: %w", err)
	}

	for _, f := range facets {
		adID, ok := adRefToID[f.AdRef]
		if !ok {
			return fmt.Errorf("unknown ad ref %d for facet", f.AdRef)
		}
		_, err := db.Exec(`
			INSERT INTO ad_facets (ad_id, "key", num, text)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (ad_id, "key") DO NOTHING`,
			adID, f.Key, f.Num, f.Text,
		)
		if err != nil {
			return fmt.Errorf("insert facet ad ref %d: %w", f.AdRef, err)
		}
	}

	for _, b := range bookmarks {
		userID, ok := userHashToID[b.UserHash]
		if !ok {
			return fmt.Errorf("unknown user hash for bookmark")
		}
		adID, ok := adRefToID[b.AdRef]
		if !ok {
			return fmt.Errorf("unknown ad ref for bookmark")
		}
		_, err := db.Exec(`
			INSERT INTO bookmarks (user_id, ad_id, bookmarked_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (user_id, ad_id) DO NOTHING`,
			userID, adID, b.BookmarkedAt,
		)
		if err != nil {
			return fmt.Errorf("insert bookmark: %w", err)
		}
	}

	for _, c := range clicks {
		userID, ok := userHashToID[c.UserHash]
		if !ok {
			return fmt.Errorf("unknown user hash for click")
		}
		adID, ok := adRefToID[c.AdRef]
		if !ok {
			return fmt.Errorf("unknown ad ref for click")
		}
		_, err := db.Exec(`
			INSERT INTO user_ad_clicks (
				ad_id, user_id, click_count, last_clicked_at
			) VALUES ($1, $2, $3, $4)
			ON CONFLICT (ad_id, user_id) DO NOTHING`,
			adID, userID, c.ClickCount, c.LastClickedAt,
		)
		if err != nil {
			return fmt.Errorf("insert click: %w", err)
		}
	}

	for _, c := range imageClicks {
		userID, ok := userHashToID[c.UserHash]
		if !ok {
			return fmt.Errorf("unknown user hash for image click")
		}
		adID, ok := adRefToID[c.AdRef]
		if !ok {
			return fmt.Errorf("unknown ad ref for image click")
		}
		_, err := db.Exec(`
			INSERT INTO user_ad_image_clicks (
				ad_id, user_id, image_index, click_count, last_clicked_at
			) VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (ad_id, user_id, image_index) DO NOTHING`,
			adID, userID, c.ImageIndex, c.ClickCount, c.LastClickedAt,
		)
		if err != nil {
			return fmt.Errorf("insert image click: %w", err)
		}
	}

	sort.Slice(conversations, func(i, j int) bool {
		return conversations[i].Ref < conversations[j].Ref
	})

	msgsByConv := make(map[int][]MessageRow, len(conversations))
	for _, m := range messages {
		msgsByConv[m.ConversationRef] = append(msgsByConv[m.ConversationRef], m)
	}
	for ref := range msgsByConv {
		sort.Slice(msgsByConv[ref], func(i, j int) bool {
			return msgsByConv[ref][i].CreatedAt.Before(msgsByConv[ref][j].CreatedAt)
		})
	}

	convRefToID := make(map[int]int, len(conversations))
	for _, c := range conversations {
		adID, ok := adRefToID[c.AdRef]
		if !ok {
			return fmt.Errorf("unknown ad ref for conversation %d", c.Ref)
		}
		ownerID, ok := userHashToID[c.OwnerHash]
		if !ok {
			return fmt.Errorf("unknown owner for conversation %d", c.Ref)
		}
		inquirerID, ok := userHashToID[c.InquirerHash]
		if !ok {
			return fmt.Errorf("unknown inquirer for conversation %d", c.Ref)
		}
		var rockThrowerID any
		if c.RockThrowerHash != nil {
			id, ok := userHashToID[*c.RockThrowerHash]
			if !ok {
				return fmt.Errorf(
					"unknown rock thrower for conversation %d", c.Ref,
				)
			}
			rockThrowerID = id
		}
		msgs := msgsByConv[c.Ref]
		journalText := buildConversationJournal(c, msgs, userHashToID)
		updatedAt := conversationUpdatedAt(c, msgs)
		var newID int
		err := db.QueryRow(`
			INSERT INTO conversations (
				ad_id, owner_id, inquirer_id,
				owner_has_unread, inquirer_has_unread,
				rock_thrower_id, rock_thrown_at, journal, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id`,
			adID, ownerID, inquirerID,
			c.OwnerHasUnread, c.InquirerHasUnread,
			rockThrowerID, c.RockThrownAt,
			"", updatedAt,
		).Scan(&newID)
		if err != nil {
			return fmt.Errorf("insert conversation ref %d: %w", c.Ref, err)
		}
		sealed, err := encryption.Seal(
			newID, journalText, config.DBEncryptionKey,
		)
		if err != nil {
			return fmt.Errorf("seal journal for conversation %d: %w", newID, err)
		}
		_, err = db.Exec(
			`UPDATE conversations SET journal = $1 WHERE id = $2`,
			sealed, newID,
		)
		if err != nil {
			return fmt.Errorf("store journal for conversation %d: %w", newID, err)
		}
		convRefToID[c.Ref] = newID
	}

	for _, o := range opinions {
		convID, ok := convRefToID[o.ConversationRef]
		if !ok {
			return fmt.Errorf(
				"unknown conversation ref %d for opinion", o.ConversationRef,
			)
		}
		_, err := db.Exec(`
			INSERT INTO rock_opinions (
				conversation_id, generated_at, summary, assessment,
				assessment_detail, resolution, reasoning
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
			ON CONFLICT (conversation_id) DO NOTHING`,
			convID, o.GeneratedAt, o.Summary, o.Assessment,
			o.AssessmentDetail, o.Resolution, o.Reasoning,
		)
		if err != nil {
			return fmt.Errorf("insert rock opinion: %w", err)
		}
	}

	if err := syncIdentitySequences(); err != nil {
		return err
	}

	imageDir := filepath.Join(fromDir, dirImages)
	uploaded := 0
	if _, err := os.Stat(imageDir); err == nil {
		err = filepath.WalkDir(imageDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(imageDir, path)
			if err != nil {
				return err
			}
			adRef, index, suffix, ok := parseRestoreImagePath(filepath.ToSlash(rel))
			if !ok {
				logger.Warn("Skipping unrecognized image", "path", rel)
				return nil
			}
			adID, ok := adRefToID[adRef]
			if !ok {
				return fmt.Errorf("unknown ad ref %d for image %s", adRef, rel)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read image %s: %w", path, err)
			}
			data, err = imgconv.ToJPEG(data, imgconv.DefaultQuality)
			if err != nil {
				return fmt.Errorf("convert image %s: %w", rel, err)
			}
			if err := store.Put(adID, index, suffix, data); err != nil {
				return fmt.Errorf("upload image %s: %w", rel, err)
			}
			uploaded++
			if verbose {
				logger.Info("Restored image",
					"ad_id", adID, "ad_ref", adRef,
					"index", index, "size", suffix)
			}
			return nil
		})
		if err != nil {
			return err
		}
	}

	uploadedAccounts, err := restoreUserAccountPictures(
		fromDir, store, userHashToID, users, verbose)
	if err != nil {
		return err
	}

	logger.Info("Restore complete",
		"dir", fromDir, "ads", len(ads), "images", uploaded,
		"account_pictures", uploadedAccounts)
	return nil
}

func resolveLocationID(loc LocationRow) (int, error) {
	var id int
	err := db.QueryRow(
		`SELECT id FROM locations WHERE raw_text = $1`, loc.RawText,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("lookup location %q: %w", loc.RawText, err)
	}
	err = db.QueryRow(`
		INSERT INTO locations (
			raw_text, city, admin_area, country, latitude, longitude
		) VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		loc.RawText, loc.City, loc.AdminArea,
		loc.Country, loc.Latitude, loc.Longitude,
	).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert location %q: %w", loc.RawText, err)
	}
	return id, nil
}

func buildConversationJournal(c ConversationRow, msgs []MessageRow,
	userHashToID map[string]int) string {
	type journalEvent struct {
		at     time.Time
		append func(string) string
	}
	var events []journalEvent

	if c.RockThrowerHash != nil && c.RockThrownAt != nil {
		throwerID := userHashToID[*c.RockThrowerHash]
		thrownAt := *c.RockThrownAt
		events = append(events, journalEvent{
			at: thrownAt,
			append: func(j string) string {
				return journal.AppendRock(j, journal.RockThrown, throwerID, "",
					thrownAt, time.UTC)
			},
		})
	}
	for _, m := range msgs {
		senderID := userHashToID[m.SenderHash]
		content := m.Content
		createdAt := m.CreatedAt
		events = append(events, journalEvent{
			at: createdAt,
			append: func(j string) string {
				return journal.AppendMessage(j, senderID, content, createdAt,
					time.UTC)
			},
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].at.Before(events[j].at)
	})

	j := ""
	for _, e := range events {
		j = e.append(j)
	}
	return j
}

func conversationUpdatedAt(c ConversationRow, msgs []MessageRow) time.Time {
	var t time.Time
	if c.RockThrownAt != nil {
		t = *c.RockThrownAt
	}
	for _, m := range msgs {
		if m.CreatedAt.After(t) {
			t = m.CreatedAt
		}
	}
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}

func syncIdentitySequences() error {
	tables := []string{
		"users", "locations", "ads", "conversations",
	}
	for _, table := range tables {
		var maxID int
		err := db.QueryRow(fmt.Sprintf(
			`SELECT COALESCE(MAX(id), 0) FROM %s`, table,
		)).Scan(&maxID)
		if err != nil {
			return fmt.Errorf("max id for %s: %w", table, err)
		}
		if maxID == 0 {
			continue
		}
		_, err = db.Exec(fmt.Sprintf(
			`SELECT setval(pg_get_serial_sequence('%s', 'id'), $1)`, table,
		), maxID)
		if err != nil {
			return fmt.Errorf("setval %s: %w", table, err)
		}
	}
	return nil
}

func prepareFreshDatabase() error {
	logger.Info("Resetting database for restore")
	if err := dbinit.Rebuild(); err != nil {
		return err
	}
	return nil
}

func parseRestoreImagePath(rel string) (adRef, index int, suffix string, ok bool) {
	parts := strings.Split(rel, "/")
	if len(parts) != 2 {
		return 0, 0, "", false
	}
	adRef, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, "", false
	}
	matches := restoreImagePattern.FindStringSubmatch(parts[1])
	if len(matches) != 4 {
		return 0, 0, "", false
	}
	index, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, "", false
	}
	return adRef, index, matches[2], true
}

var restoreUserAccountHash = regexp.MustCompile(`^[0-9a-f]{64}$`)

func restoreUserAccountPictures(fromDir string, store imagestore.Store,
	userHashToID map[string]int, users []UserRow, verbose bool) (int, error) {
	accountDir := filepath.Join(fromDir, dirUserAccounts)
	uploaded := 0
	restored := make(map[string]bool)
	if _, err := os.Stat(accountDir); err == nil {
		err = filepath.WalkDir(accountDir, func(
			path string, d os.DirEntry, walkErr error,
		) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(accountDir, path)
			if err != nil {
				return err
			}
			nameHash, ok := parseRestoreUserAccountPath(
				filepath.ToSlash(rel))
			if !ok {
				logger.Warn("Skipping unrecognized account picture",
					"path", rel)
				return nil
			}
			userID, ok := userHashToID[nameHash]
			if !ok {
				return fmt.Errorf(
					"unknown user hash for account picture %s", rel)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}
			data, err = imgconv.ToJPEG(data, imgconv.DefaultQuality)
			if err != nil {
				return fmt.Errorf("convert %s: %w", rel, err)
			}
			if err := store.PutUserAccount(userID, data); err != nil {
				return fmt.Errorf("upload %s: %w", rel, err)
			}
			restored[nameHash] = true
			uploaded++
			if verbose {
				logger.Info("Restored account picture",
					"user_id", userID, "name_hash", nameHash)
			}
			return nil
		})
		if err != nil {
			return 0, err
		}
	}
	for _, u := range users {
		if u.HasAccountPicture == 0 || restored[u.NameHash] {
			continue
		}
		logger.Warn("Account picture missing in archive",
			"name_hash", u.NameHash)
	}
	return uploaded, nil
}

func parseRestoreUserAccountPath(rel string) (nameHash string, ok bool) {
	parts := strings.Split(rel, "/")
	if len(parts) != 2 || parts[1] != "account.jpg" {
		return "", false
	}
	if !restoreUserAccountHash.MatchString(parts[0]) {
		return "", false
	}
	return parts[0], true
}
