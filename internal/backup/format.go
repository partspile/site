package backup

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/partspile/site/internal/config"
	"github.com/pgvector/pgvector-go"
)

const (
	fileManifest          = "manifest.json"
	fileUsers             = "users.json"
	fileLocations         = "locations.json"
	fileAds               = "ads.json"
	fileAdFacets          = "ad_facets.json"
	fileBookmarks         = "bookmarks.json"
	fileUserAdClicks      = "user_ad_clicks.json"
	fileUserAdImageClicks = "user_ad_image_clicks.json"
	fileConversations     = "conversations.json"
	fileMessages          = "messages.json"
	fileRockOpinions      = "rock_opinions.json"
	dirImages             = "images"
	dirUserAccounts       = "user_accounts"
)

type Manifest struct {
	CreatedAt time.Time `json:"created_at"`
	Counts    Counts    `json:"counts"`
}

type Counts struct {
	Users             int `json:"users"`
	Locations         int `json:"locations"`
	Ads               int `json:"ads"`
	AdFacets          int `json:"ad_facets"`
	Bookmarks         int `json:"bookmarks"`
	UserAdClicks      int `json:"user_ad_clicks"`
	UserAdImageClicks int `json:"user_ad_image_clicks"`
	Conversations     int `json:"conversations"`
	Messages          int `json:"messages"`
	RockOpinions      int `json:"rock_opinions"`
	Images            int `json:"images"`
	AccountPictures   int `json:"account_pictures"`
	Embeddings        int `json:"embeddings"`
}

type B64 []byte

func (b B64) MarshalJSON() ([]byte, error) {
	if len(b) == 0 {
		return json.Marshal("")
	}
	return json.Marshal(base64.StdEncoding.EncodeToString(b))
}

func (b *B64) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		*b = nil
		return nil
	}
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}
	*b = decoded
	return nil
}

type UserRow struct {
	EncryptUserID     int        `json:"encrypt_user_id"`
	CreatedAt         time.Time  `json:"created_at" db:"created_at"`
	DeletedAt         *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
	IsAdmin           int        `json:"is_admin" db:"is_admin"`
	EncryptedName     B64        `json:"encrypted_name" db:"encrypted_name"`
	NameNonce         B64        `json:"name_nonce" db:"name_nonce"`
	NameHash          string     `json:"name_hash" db:"name_hash"`
	PasswordHash      string     `json:"password_hash" db:"password_hash"`
	PasswordSalt      string     `json:"password_salt" db:"password_salt"`
	PasswordAlgo      string     `json:"password_algo" db:"password_algo"`
	EncryptedPhone    B64        `json:"encrypted_phone" db:"encrypted_phone"`
	PhoneNonce        B64        `json:"phone_nonce" db:"phone_nonce"`
	PhoneHash         string     `json:"phone_hash" db:"phone_hash"`
	PhoneVerified     int        `json:"phone_verified" db:"phone_verified"`
	SMSOptedOut       int        `json:"sms_opted_out" db:"sms_opted_out"`
	LastSMSSentAt     *time.Time `json:"last_sms_sent_at,omitempty" db:"last_sms_sent_at"`
	HasAccountPicture int        `json:"has_account_picture" db:"has_account_picture"`
	AccountPictureURL *string    `json:"account_picture_url,omitempty" db:"account_picture_url"`
}

type userDBRow struct {
	ID                int        `db:"id"`
	CreatedAt         time.Time  `db:"created_at"`
	DeletedAt         *time.Time `db:"deleted_at"`
	IsAdmin           int        `db:"is_admin"`
	EncryptedName     []byte     `db:"encrypted_name"`
	NameNonce         []byte     `db:"name_nonce"`
	NameHash          string     `db:"name_hash"`
	PasswordHash      string     `db:"password_hash"`
	PasswordSalt      string     `db:"password_salt"`
	PasswordAlgo      string     `db:"password_algo"`
	EncryptedPhone    []byte     `db:"encrypted_phone"`
	PhoneNonce        []byte     `db:"phone_nonce"`
	PhoneHash         string     `db:"phone_hash"`
	PhoneVerified     int        `db:"phone_verified"`
	SMSOptedOut       int        `db:"sms_opted_out"`
	LastSMSSentAt     *time.Time `db:"last_sms_sent_at"`
	HasAccountPicture int        `db:"has_account_picture"`
	AccountPictureURL *string    `db:"account_picture_url"`
}

func userRowFromDB(r userDBRow) UserRow {
	return UserRow{
		EncryptUserID:     r.ID,
		CreatedAt:         r.CreatedAt,
		DeletedAt:         r.DeletedAt,
		IsAdmin:           r.IsAdmin,
		EncryptedName:     B64(r.EncryptedName),
		NameNonce:         B64(r.NameNonce),
		NameHash:          r.NameHash,
		PasswordHash:      r.PasswordHash,
		PasswordSalt:      r.PasswordSalt,
		PasswordAlgo:      r.PasswordAlgo,
		EncryptedPhone:    B64(r.EncryptedPhone),
		PhoneNonce:        B64(r.PhoneNonce),
		PhoneHash:         r.PhoneHash,
		PhoneVerified:     r.PhoneVerified,
		SMSOptedOut:       r.SMSOptedOut,
		LastSMSSentAt:     r.LastSMSSentAt,
		HasAccountPicture: r.HasAccountPicture,
		AccountPictureURL: r.AccountPictureURL,
	}
}

type LocationRow struct {
	RawText   string  `json:"raw_text" db:"raw_text"`
	City      string  `json:"city" db:"city"`
	AdminArea string  `json:"admin_area" db:"admin_area"`
	Country   string  `json:"country" db:"country"`
	Latitude  float64 `json:"latitude" db:"latitude"`
	Longitude float64 `json:"longitude" db:"longitude"`
}

type AdRow struct {
	Ref            int             `json:"ref"`
	CategoryID     int             `json:"category_id" db:"category_id"`
	Title          string          `json:"title" db:"title"`
	Description    string          `json:"description" db:"description"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	ExpiresAt      time.Time       `json:"expires_at" db:"expires_at"`
	InactiveAt     *time.Time      `json:"inactive_at,omitempty" db:"inactive_at"`
	DeletedAt      *time.Time      `json:"deleted_at,omitempty" db:"deleted_at"`
	OwnerHash      string          `json:"owner_hash"`
	ImageCount     int             `json:"image_count" db:"image_count"`
	LocationRaw    *string         `json:"location_raw,omitempty"`
	Tags           string          `json:"tags" db:"tags"`
	Embedding      B64             `json:"embedding,omitempty"`
	VectorMetadata json.RawMessage `json:"vector_metadata,omitempty"`
}

type adDBRow struct {
	ID             int        `db:"id"`
	CategoryID     int        `db:"category_id"`
	Title          string     `db:"title"`
	Description    string     `db:"description"`
	CreatedAt      time.Time  `db:"created_at"`
	ExpiresAt      time.Time  `db:"expires_at"`
	InactiveAt     *time.Time `db:"inactive_at"`
	DeletedAt      *time.Time `db:"deleted_at"`
	UserID         int        `db:"user_id"`
	ImageCount     int        `db:"image_count"`
	LocationID     *int       `db:"location_id"`
	Tags           string     `db:"tags"`
	EmbeddingText  *string    `db:"embedding_text"`
	VectorMetadata *string    `db:"vector_metadata"`
}

func encodeEmbedding(text string) (B64, error) {
	var v pgvector.Vector
	if err := v.Parse(text); err != nil {
		return nil, fmt.Errorf("parse embedding: %w", err)
	}
	buf, err := v.EncodeBinary(nil)
	if err != nil {
		return nil, fmt.Errorf("encode embedding: %w", err)
	}
	return B64(buf), nil
}

func decodeEmbedding(b B64) ([]float32, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var v pgvector.Vector
	if err := v.DecodeBinary(b); err != nil {
		return nil, fmt.Errorf("decode embedding: %w", err)
	}
	slice := v.Slice()
	if len(slice) != config.OllamaEmbeddingDimensions {
		return nil, fmt.Errorf(
			"embedding dim %d, want %d",
			len(slice), config.OllamaEmbeddingDimensions,
		)
	}
	return slice, nil
}

type AdFacetRow struct {
	AdRef int     `json:"ad_ref"`
	Key   string  `json:"key" db:"key"`
	Num   *int    `json:"num,omitempty" db:"num"`
	Text  *string `json:"text,omitempty" db:"text"`
}

type adFacetDBRow struct {
	AdID int     `db:"ad_id"`
	Key  string  `db:"key"`
	Num  *int    `db:"num"`
	Text *string `db:"text"`
}

type BookmarkRow struct {
	UserHash     string    `json:"user_hash"`
	AdRef        int       `json:"ad_ref"`
	BookmarkedAt time.Time `json:"bookmarked_at" db:"bookmarked_at"`
}

type bookmarkDBRow struct {
	UserID       int       `db:"user_id"`
	AdID         int       `db:"ad_id"`
	BookmarkedAt time.Time `db:"bookmarked_at"`
}

type UserAdClickRow struct {
	AdRef         int       `json:"ad_ref"`
	UserHash      string    `json:"user_hash"`
	ClickCount    int       `json:"click_count" db:"click_count"`
	LastClickedAt time.Time `json:"last_clicked_at" db:"last_clicked_at"`
}

type userAdClickDBRow struct {
	AdID          int       `db:"ad_id"`
	UserID        int       `db:"user_id"`
	ClickCount    int       `db:"click_count"`
	LastClickedAt time.Time `db:"last_clicked_at"`
}

type UserAdImageClickRow struct {
	AdRef         int       `json:"ad_ref"`
	UserHash      string    `json:"user_hash"`
	ImageIndex    int       `json:"image_index" db:"image_index"`
	ClickCount    int       `json:"click_count" db:"click_count"`
	LastClickedAt time.Time `json:"last_clicked_at" db:"last_clicked_at"`
}

type userAdImageClickDBRow struct {
	AdID          int       `db:"ad_id"`
	UserID        int       `db:"user_id"`
	ImageIndex    int       `db:"image_index"`
	ClickCount    int       `db:"click_count"`
	LastClickedAt time.Time `db:"last_clicked_at"`
}

type ConversationRow struct {
	Ref               int        `json:"ref"`
	AdRef             int        `json:"ad_ref"`
	OwnerHash         string     `json:"owner_hash"`
	InquirerHash      string     `json:"inquirer_hash"`
	OwnerHasUnread    int        `json:"owner_has_unread" db:"owner_has_unread"`
	InquirerHasUnread int        `json:"inquirer_has_unread" db:"inquirer_has_unread"`
	RockThrowerHash   *string    `json:"rock_thrower_hash,omitempty"`
	RockThrownAt      *time.Time `json:"rock_thrown_at,omitempty" db:"rock_thrown_at"`
}

type conversationDBRow struct {
	ID                int        `db:"id"`
	AdID              int        `db:"ad_id"`
	OwnerID           int        `db:"owner_id"`
	InquirerID        int        `db:"inquirer_id"`
	OwnerHasUnread    int        `db:"owner_has_unread"`
	InquirerHasUnread int        `db:"inquirer_has_unread"`
	RockThrowerID     *int       `db:"rock_thrower_id"`
	RockThrownAt      *time.Time `db:"rock_thrown_at"`
	Journal           string     `db:"journal"`
}

type MessageRow struct {
	ConversationRef int       `json:"conversation_ref"`
	SenderHash      string    `json:"sender_hash"`
	Content         string    `json:"content" db:"content"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type messageDBRow struct {
	ID             int       `db:"id"`
	ConversationID int       `db:"conversation_id"`
	SenderID       int       `db:"sender_id"`
	Content        string    `db:"content"`
	CreatedAt      time.Time `db:"created_at"`
}

type RockOpinionRow struct {
	ConversationRef  int       `json:"conversation_ref"`
	GeneratedAt      time.Time `json:"generated_at" db:"generated_at"`
	Summary          string    `json:"summary" db:"summary"`
	Assessment       int       `json:"assessment" db:"assessment"`
	AssessmentDetail string    `json:"assessment_detail" db:"assessment_detail"`
	Resolution       string    `json:"resolution" db:"resolution"`
	Reasoning        string    `json:"reasoning" db:"reasoning"`
}

type rockOpinionDBRow struct {
	ConversationID   int       `db:"conversation_id"`
	GeneratedAt      time.Time `db:"generated_at"`
	Summary          string    `db:"summary"`
	Assessment       int       `db:"assessment"`
	AssessmentDetail string    `db:"assessment_detail"`
	Resolution       string    `db:"resolution"`
	Reasoning        string    `db:"reasoning"`
}
