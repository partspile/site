package ui

import (
	"fmt"
	"strconv"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func AdminDashboardPage(stats QueueStats,
	entries []SMSQueueEntry) []g.Node {
	return []g.Node{
		pageTitle("Admin Dashboard"),
		AdminDashboardContainerWithQueue("sms-queue", stats, entries),
	}
}

func AdminDashboardContainer(activeTab string) g.Node {
	return Div(
		ID("admin-dashboard-container"),
		Class("space-y-6 mt-6"),
		AdminTabs(activeTab),
		AdminContent(activeTab),
	)
}

func AdminDashboardContainerWithClicks(activeTab string,
	clicks ClickAdminData) g.Node {
	return Div(
		ID("admin-dashboard-container"),
		Class("space-y-6 mt-6"),
		AdminTabs(activeTab),
		AdminContentWithClicks(activeTab, clicks),
	)
}

func AdminContentWithClicks(activeTab string, clicks ClickAdminData) g.Node {
	return Div(
		ID("admin-content"),
		g.If(activeTab == "sms-queue",
			SMSQueueTab(QueueStats{}, []SMSQueueEntry{})),
		g.If(activeTab == "embeddings", EmbeddingsTab(EmbeddingAdminData{})),
		g.If(activeTab == "clicks", ClicksTab(clicks)),
		g.If(activeTab == "turnstile", AdminTurnstileTab()),
		g.If(activeTab == "settings", AdminSettingsTab()),
	)
}

func AdminDashboardContainerWithEmbeddings(activeTab string,
	emb EmbeddingAdminData) g.Node {
	return Div(
		ID("admin-dashboard-container"),
		Class("space-y-6 mt-6"),
		AdminTabs(activeTab),
		AdminContentWithEmbeddings(activeTab, emb),
	)
}

func AdminContentWithEmbeddings(activeTab string,
	emb EmbeddingAdminData) g.Node {
	return Div(
		ID("admin-content"),
		g.If(activeTab == "sms-queue",
			SMSQueueTab(QueueStats{}, []SMSQueueEntry{})),
		g.If(activeTab == "embeddings", EmbeddingsTab(emb)),
		g.If(activeTab == "clicks", ClicksTab(ClickAdminData{})),
		g.If(activeTab == "turnstile", AdminTurnstileTab()),
		g.If(activeTab == "settings", AdminSettingsTab()),
	)
}

func AdminDashboardContainerWithQueue(activeTab string, queueStats QueueStats,
	queueEntries []SMSQueueEntry) g.Node {
	return Div(
		ID("admin-dashboard-container"),
		Class("space-y-6 mt-6"),
		AdminTabs(activeTab),
		AdminContentWithQueue(activeTab, queueStats, queueEntries),
	)
}

func AdminTabs(activeTab string) g.Node {
	return Div(
		ID("admin-tabs"),
		Class("border-b border-zinc-200 dark:border-zinc-700"),
		Div(
			Class("flex space-x-8"),
			adminTab("SMS Queue", "sms-queue", activeTab == "sms-queue"),
			adminTab("Embeddings", "embeddings", activeTab == "embeddings"),
			adminTab("Clicks", "clicks", activeTab == "clicks"),
			adminTab("Turnstile", "turnstile", activeTab == "turnstile"),
			adminTab("Settings", "settings", activeTab == "settings"),
		),
	)
}

func adminTab(name, tabID string, active bool) g.Node {
	var classes string
	if active {
		classes = "border-b-2 border-blue-500 text-blue-600 dark:text-blue-400 py-4 px-1 text-sm font-medium"
	} else {
		classes = "border-b-2 border-transparent text-zinc-500 hover:text-zinc-700 hover:border-zinc-300 dark:text-zinc-400 dark:hover:text-zinc-300 py-4 px-1 text-sm font-medium"
	}

	href := fmt.Sprintf("/admin/tab/%s", tabID)

	return A(
		Href(href),
		hx.Get(href),
		hx.Target("#admin-dashboard-container"),
		hx.Swap("outerHTML"),
		Class(classes),
		g.Text(name),
	)
}

func AdminContent(activeTab string) g.Node {
	return Div(
		ID("admin-content"),
		g.If(activeTab == "sms-queue",
			SMSQueueTab(QueueStats{}, []SMSQueueEntry{})),
		g.If(activeTab == "embeddings", EmbeddingsTab(EmbeddingAdminData{})),
		g.If(activeTab == "clicks", ClicksTab(ClickAdminData{})),
		g.If(activeTab == "turnstile", AdminTurnstileTab()),
		g.If(activeTab == "settings", AdminSettingsTab()),
	)
}

func AdminContentWithQueue(activeTab string, queueStats QueueStats,
	queueEntries []SMSQueueEntry) g.Node {
	return Div(
		ID("admin-content"),
		g.If(activeTab == "sms-queue",
			SMSQueueTab(queueStats, queueEntries)),
		g.If(activeTab == "embeddings", EmbeddingsTab(EmbeddingAdminData{})),
		g.If(activeTab == "clicks", ClicksTab(ClickAdminData{})),
		g.If(activeTab == "turnstile", AdminTurnstileTab()),
		g.If(activeTab == "settings", AdminSettingsTab()),
	)
}

func SMSQueueTab(stats QueueStats, entries []SMSQueueEntry) g.Node {
	return Div(
		ID("sms-queue-tab"),
		Class("mt-4"),
		hx.Get("/admin/tab/sms-queue"),
		hx.Target("#admin-dashboard-container"),
		hx.Swap("outerHTML"),
		hx.Trigger("every 5s"),
		hx.Include("#sms-queue-status-filter"),
		SMSQueueStats(stats),
		SMSQueueTableWithEntries(entries),
	)
}

func SMSQueueStats(stats QueueStats) g.Node {
	return Div(
		Class("mb-4 grid grid-cols-3 gap-4"),
		Div(
			Class("bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4"),
			Div(
				Class("text-sm font-medium text-yellow-800 dark:text-yellow-200"),
				g.Text("Pending"),
			),
			Div(
				ID("sms-queue-pending-count"),
				Class("text-2xl font-bold text-yellow-900 dark:text-yellow-100"),
				g.Text(strconv.Itoa(stats.Pending)),
			),
		),
		Div(
			Class("bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4"),
			Div(
				Class("text-sm font-medium text-green-800 dark:text-green-200"),
				g.Text("Processed"),
			),
			Div(
				ID("sms-queue-processed-count"),
				Class("text-2xl font-bold text-green-900 dark:text-green-100"),
				g.Text(strconv.Itoa(stats.Processed)),
			),
		),
		Div(
			Class("bg-gray-50 dark:bg-gray-900/20 border border-gray-200 dark:border-gray-800 rounded-lg p-4"),
			Div(
				Class("text-sm font-medium text-gray-800 dark:text-gray-200"),
				g.Text("Suppressed"),
			),
			Div(
				ID("sms-queue-suppressed-count"),
				Class("text-2xl font-bold text-gray-900 dark:text-gray-100"),
				g.Text(strconv.Itoa(stats.Suppressed)),
			),
		),
	)
}

func SMSQueueTableWithEntries(entries []SMSQueueEntry) g.Node {
	return Div(
		ID("sms-queue-table"),
		Class("bg-white dark:bg-zinc-800 rounded-lg shadow overflow-hidden"),
		SMSQueueFilters(),
		SMSQueueTableHeader(),
		SMSQueueRows(entries),
	)
}

func SMSQueueFilters() g.Node {
	return Div(
		Class("p-4 border-b border-zinc-200 dark:border-zinc-700"),
		Select(
			ID("sms-queue-status-filter"),
			Name("status"),
			Class("px-3 py-2 border border-zinc-300 dark:border-zinc-600 rounded-md bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-200 text-sm"),
			hx.Get("/admin/tab/sms-queue"),
			hx.Target("#admin-dashboard-container"),
			hx.Swap("outerHTML"),
			hx.Trigger("change"),
			hx.Include("#sms-queue-status-filter"),
			Option(Value("all"), Selected(), g.Text("All Statuses")),
			Option(Value("pending"), g.Text("Pending")),
			Option(Value("processed"), g.Text("Processed")),
			Option(Value("suppressed"), g.Text("Suppressed")),
		),
	)
}

func SMSQueueTableHeader() g.Node {
	return Div(
		Class("grid grid-cols-5 gap-2 bg-zinc-50 dark:bg-zinc-800 px-4 py-2 border-b border-zinc-200 dark:border-zinc-700 text-xs"),
		Div(
			Class("font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider"),
			g.Text("ID"),
		),
		Div(
			Class("font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider"),
			g.Text("Conversation"),
		),
		Div(
			Class("font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider"),
			g.Text("Status"),
		),
		Div(
			Class("font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider"),
			g.Text("Created At"),
		),
		Div(
			Class("font-medium text-zinc-500 dark:text-zinc-400 uppercase tracking-wider"),
			g.Text("Processed At"),
		),
	)
}

func SMSQueueRows(entries []SMSQueueEntry) g.Node {
	rows := make([]g.Node, len(entries))
	for i, entry := range entries {
		rows[i] = SMSQueueRow(entry)
	}
	return Div(
		ID("sms-queue-rows"),
		Class("divide-y divide-zinc-200 dark:divide-zinc-700"),
		g.Group(rows),
	)
}

func SMSQueueRow(entry SMSQueueEntry) g.Node {
	var statusClass string
	switch entry.Status {
	case "pending":
		statusClass = "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-200"
	case "processed":
		statusClass = "bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-200"
	case "suppressed":
		statusClass = "bg-gray-100 dark:bg-gray-900/30 text-gray-800 dark:text-gray-200"
	default:
		statusClass = "bg-zinc-100 dark:bg-zinc-900/30 text-zinc-800 dark:text-zinc-200"
	}

	createdAtStr := entry.CreatedAt
	var processedAtStr string
	if entry.ProcessedAt != nil && *entry.ProcessedAt != "" {
		processedAtStr = *entry.ProcessedAt
	} else {
		processedAtStr = "—"
	}

	return Div(
		Class("grid grid-cols-5 gap-2 px-4 py-2 text-xs text-zinc-900 dark:text-zinc-200"),
		Div(
			g.Text(strconv.Itoa(entry.ID)),
		),
		Div(
			g.Text(entry.AdTitle),
		),
		Div(
			Span(
				Class("px-2 py-1 rounded text-xs font-medium "+statusClass),
				g.Text(entry.Status),
			),
		),
		Div(
			g.Text(createdAtStr),
		),
		Div(
			g.Text(processedAtStr),
		),
	)
}

// SMSQueueEntry represents a queue entry for UI rendering
type SMSQueueEntry struct {
	ID          int
	AdTitle     string
	Status      string
	CreatedAt   string
	ProcessedAt *string
}

func SMSQueueEntriesFrom(entries []SMSQueueEntryInput) []SMSQueueEntry {
	result := make([]SMSQueueEntry, len(entries))
	for i, e := range entries {
		createdAtStr := e.CreatedAt.Format("2006-01-02 15:04:05")
		var processedAtStr *string
		if e.ProcessedAt != nil {
			s := e.ProcessedAt.Format("2006-01-02 15:04:05")
			processedAtStr = &s
		}
		result[i] = SMSQueueEntry{
			ID:          e.ID,
			AdTitle:     e.AdTitle,
			Status:      e.Status,
			CreatedAt:   createdAtStr,
			ProcessedAt: processedAtStr,
		}
	}
	return result
}

func AdminSettingsTab() g.Node {
	return Div(
		Class("mt-4 p-6 bg-white dark:bg-zinc-800 rounded-lg shadow"),
		H2(
			Class("text-xl font-semibold text-zinc-900 dark:text-zinc-200 mb-4"),
			g.Text("Settings"),
		),
		P(
			Class("text-zinc-600 dark:text-zinc-400"),
			g.Text("This is a dummy settings tab for testing purposes."),
		),
	)
}
