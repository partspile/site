package ui

import (
	"time"

	g "maragu.dev/gomponents"
)

// UserProfileData holds presentation fields for a user profile or summary.
type UserProfileData struct {
	ID                int
	Name              string
	MemberSince       string
	ActiveAdCount     int
	UserFlagCount     int
	HasAccountPicture bool
	AccountPictureURL string
	ShowShare         bool
	IsOwn             bool // viewer is this profile's owner
}

// AdHistoryEntry holds one edit-history block on the ad detail page.
type AdHistoryEntry struct {
	Header       string
	Body         string
	ImageIndices []int
}

// AdDetail holds presentation fields for a single ad page.
type AdDetail struct {
	ID                  int
	OwnerID             int
	ImageCount          int
	PriceDisplay        string
	HasPrice            bool
	Title               string
	Location            string
	DescriptionOriginal string
	DescriptionHistory  []AdHistoryEntry
	// ShowLoginForDetails is true when edit history exists but is hidden
	// until the viewer logs in.
	ShowLoginForDetails bool
	CreatedAt           time.Time
	ExpiresAt           time.Time
	Bookmarked          bool
	Active              bool
	Inactive            bool
	Deleted             bool
	IsTest              bool
	Reachable           bool
	FlagCount           int
	FacetLabel          string
	FacetDetails        []string // formal facets not shown in title or price row
	Tags                []string
}

// AdFlyerData holds presentation fields for the print-only ad flyer.
type AdFlyerData struct {
	ID           int
	ImageCount   int
	PriceDisplay string
	HasPrice     bool
	Title        string
	Location     string
	Description  string
	FacetLabel   string
	FacetDetails []string
	Tags         []string
	AdURL        string
}

// AdCard holds presentation fields for a listing card.
type AdCard struct {
	ID           int
	PriceDisplay string
	HasPrice     bool
	Title        string
	Location     string
	CreatedAt    time.Time
	ImageCount   int
	Active       bool
	Bookmarked   bool
	FlagCount    int
	FacetLabel   string
}

// ConversationModalData holds presentation fields for a conversation modal.
type ConversationModalData struct {
	ConversationID    int
	AdID              int
	OwnerID           int
	InquirerID        int
	CurrentUserID     int
	InquirerFlagCount int
	OwnerFlagCount    int
	AdTitle           string
	OwnerName         string
	InquirerName      string
	OwnerDeleted      bool
	InquirerDeleted   bool
	CSRFToken         string
	CanPost           bool
	HasRaisedFlag     bool
	CanRaiseFlag      bool
	MessageNodes      []g.Node
	FlagRaiserID      *int
	TargetModalID     string
	// DisabledInputPlaceholder overrides the closed send-field placeholder.
	DisabledInputPlaceholder string
}

// ConversationListItemData holds presentation fields for a messages list row.
// LastMessageAt and UpdatedAt are in the viewer's timezone when set.
type ConversationListItemData struct {
	ConversationID     int
	AdID               int
	OwnerID            int
	InquirerID         int
	CurrentUserID      int
	AdTitle            string
	LastMessageContent string
	OtherUserName      string
	LastMessageAt      *time.Time
	UpdatedAt          time.Time
	HasUnread          bool
	FlagCount          int
	OtherUserFlagCount int
	OtherUserDeleted   bool
}

// CategoryOption holds presentation fields for a category picker item.
type CategoryOption struct {
	ID        int
	Name      string
	ImageFile string
}

// SMSQueueEntryInput holds raw queue entry fields for UI formatting.
type SMSQueueEntryInput struct {
	ID          int
	AdTitle     string
	Status      string
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

// MessageItemData holds presentation fields for a single chat message.
// CreatedAt is in the viewer's timezone.
type MessageItemData struct {
	SenderID      int
	CurrentUserID int
	Content       string
	CreatedAt     time.Time
}

// FlagEventKind identifies a flag journal entry type.
type FlagEventKind int

const (
	FlagEventThrown FlagEventKind = iota
	FlagEventLowern
)

// FlagEventData holds presentation fields for a flag journal timeline entry.
// EventAt is in the viewer's timezone.
type FlagEventData struct {
	ConversationID     int
	ThrowerID          int
	CurrentUserID      int
	Kind               FlagEventKind
	EventAt            time.Time
	OwnerID            int
	InquirerID         int
	OwnerName          string
	InquirerName       string
	Reason             string
	ShowAssessmentHint bool
}

// CloseEventData holds presentation fields for an ad/account close journal entry.
// EventAt is in the viewer's timezone.
type CloseEventData struct {
	Text    string
	EventAt time.Time
}

// QueueStats holds SMS queue counts for the admin dashboard.
type QueueStats struct {
	Pending    int
	Processed  int
	Suppressed int
}

// EmbeddingAdminData holds presentation fields for the embeddings admin tab.
type EmbeddingAdminData struct {
	EmbedderProvider string
	EmbedderModel    string
	EmbeddedCount    int
	MissingCount     int
	QueueDepth       int
	Caches           []EmbeddingCachePanel
	MissingAds       []MissingEmbeddingRow
	CategoryID       int
	SiteCategoryID   int
	Categories       []CategoryOption
	UserID           int
	Users            []UserOption
	UserActivities   []EmbeddingActivityRow
	SiteActivities   []EmbeddingActivityRow
}

// UserOption holds presentation fields for an admin user picker item.
type UserOption struct {
	ID   int
	Name string
}

// EmbeddingActivityRow holds one weighted activity used to build an embedding.
type EmbeddingActivityRow struct {
	AdID         int
	AdTitle      string
	ActivityType string
	Weight       float32
	Timestamp    string
}

// EmbeddingCachePanel holds cache metrics for one embedding cache tier.
type EmbeddingCachePanel struct {
	Name       string
	Hits       int64
	Misses     int64
	HitRatePct float64
	ItemCount  int64
	MemoryKB   float64
}

// MissingEmbeddingRow holds one ad awaiting vector indexing.
type MissingEmbeddingRow struct {
	AdID         int
	Title        string
	CategoryName string
}

// ClickAdminData holds presentation fields for the clicks admin tab.
type ClickAdminData struct {
	UsersWithClicks int
	AdsClicked      int
	AdDetailViews   int
	ImageNavClicks  int
	ActiveLast7Days int
	TopAds          []ClickAdRow
	TopImages       []ClickImageRow
}

// ClickAdRow is one row in the top-ads table.
type ClickAdRow struct {
	AdID         int
	Title        string
	CategoryName string
	UserCount    int
	AdViews      int
	ImageClicks  int
	LastActivity string
}

// ClickImageRow is one row in the top-images table.
type ClickImageRow struct {
	AdID       int
	Title      string
	ImageIndex int
	UserCount  int
	Clicks     int
	LastClick  string
}

// FunStatsMonth is one month on the public fun stats graphs.
type FunStatsMonth struct {
	Label              string
	RegisteredUsers    int
	UsersWithActiveAds int
	ActiveAds          int
}

// FunStatsData holds presentation fields for /funstats.
type FunStatsData struct {
	Months []FunStatsMonth
}
