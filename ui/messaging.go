package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/messaging"
	"github.com/parts-pile/site/user"
)

// MessagesPage renders the main messages page
func MessagesPage(currentUser *user.User, conversations []messaging.Conversation) g.Node {
	return Page(
		"Messages",
		currentUser,
		"/messages",
		[]g.Node{
			PageHeader("Messages"),
			ContentContainer(
				g.If(len(conversations) == 0,
					Div(
						Class("text-center py-8"),
						P(Class("text-gray-500 text-lg"), g.Text("No conversations yet.")),
						P(Class("text-gray-400"), g.Text("Start a conversation by clicking the message button on any ad.")),
					),
				),
				g.If(len(conversations) > 0,
					ConversationsList(conversations, currentUser.ID),
				),
			),
		},
	)
}

// MessagesPageWithExpanded renders the messages page with a specific conversation pre-expanded
func MessagesPageWithExpanded(currentUser *user.User, conversations []messaging.Conversation, expandID string) g.Node {
	// Find the conversation to expand
	var expandedConversation *messaging.Conversation
	var expandedMessages []messaging.Message

	for _, conv := range conversations {
		if fmt.Sprintf("%d", conv.ID) == expandID {
			expandedConversation = &conv
			break
		}
	}

	// If conversation found, get its messages
	if expandedConversation != nil {
		messages, err := messaging.GetMessages(expandedConversation.ID)
		if err == nil {
			expandedMessages = messages
		}
	}

	return Page(
		"Messages",
		currentUser,
		"/messages",
		[]g.Node{
			PageHeader("Messages"),
			ContentContainer(
				g.If(len(conversations) == 0,
					Div(
						Class("text-center py-8"),
						P(Class("text-gray-500 text-lg"), g.Text("No conversations yet.")),
						P(Class("text-gray-400"), g.Text("Start a conversation by clicking the message button on any ad.")),
					),
				),
				g.If(len(conversations) > 0,
					ConversationsListWithExpanded(conversations, currentUser, expandedConversation, expandedMessages),
				),
			),
		},
	)
}

// ConversationsList renders the list of conversations
func ConversationsList(conversations []messaging.Conversation, currentUserID int) g.Node {
	var conversationNodes []g.Node
	for _, conv := range conversations {
		conversationNodes = append(conversationNodes, ConversationListItem(conv, currentUserID))
	}

	return Div(
		// SSE connection for real-time conversation updates
		g.Attr("hx-ext", "sse"),
		g.Attr("sse-connect", "/messages/sse"),
		Class("divide-y divide-gray-200"),
		g.Group(conversationNodes),
	)
}

// ConversationsListWithExpanded renders the list of conversations with one pre-expanded
func ConversationsListWithExpanded(conversations []messaging.Conversation, currentUser *user.User, expandedConversation *messaging.Conversation, expandedMessages []messaging.Message) g.Node {
	var conversationNodes []g.Node
	for _, conv := range conversations {
		if expandedConversation != nil && conv.ID == expandedConversation.ID {
			// Render the expanded conversation
			conversationNodes = append(conversationNodes, ExpandedConversation(*currentUser, conv, expandedMessages))
		} else {
			// Render the collapsed conversation
			conversationNodes = append(conversationNodes, ConversationListItem(conv, currentUser.ID))
		}
	}

	return Div(
		Class("divide-y divide-gray-200"),
		g.Group(conversationNodes),
	)
}

// ConversationListItem renders a single conversation item
func ConversationListItem(conv messaging.Conversation, currentUserID int) g.Node {
	// Determine the other participant's name
	otherUserName := conv.User1Name
	if currentUserID == conv.User1ID {
		otherUserName = conv.User2Name
	}

	// Format the time - use last message time if available, otherwise use conversation creation time
	var timeStr string
	if !conv.LastMessageAt.IsZero() {
		timeStr = formatAdAge(conv.LastMessageAt)
	} else {
		timeStr = formatAdAge(conv.CreatedAt)
	}

	// Show unread count if any
	unreadBadge := g.Node(nil)
	if conv.UnreadCount > 0 {
		unreadBadge = Span(
			Class("bg-blue-500 text-white text-xs font-bold px-2 py-1 rounded-full"),
			g.Text(fmt.Sprintf("%d", conv.UnreadCount)),
		)
	}

	return Div(
		ID(fmt.Sprintf("conversation-%d", conv.ID)),
		// SSE trigger to update when new message arrives
		hx.Get(fmt.Sprintf("/messages/%d/sse-update", conv.ID)),
		hx.Trigger("sse:new_message"),
		hx.Target(fmt.Sprintf("#conversation-%d", conv.ID)),
		hx.Swap("outerHTML"),
		Div(
			Class("py-3 px-4 hover:bg-gray-50 transition-colors cursor-pointer"),
			hx.Get(fmt.Sprintf("/messages/%d/expand", conv.ID)),
			hx.Target(fmt.Sprintf("#conversation-%d", conv.ID)),
			hx.Swap("outerHTML"),

			hx.On("htmx:beforeRequest", fmt.Sprintf("console.log('Expanding conversation %d');", conv.ID)),
			hx.On("htmx:afterRequest", fmt.Sprintf("console.log('Conversation %d expanded');", conv.ID)),
			Div(
				Class("grid grid-cols-3 gap-4 items-center"),
				Div(
					Class("flex items-center gap-2 min-w-0"),
					Span(
						g.If(conv.IsUnread, Class("text-sm font-bold text-gray-900")),
						g.If(!conv.IsUnread, Class("text-sm font-medium text-gray-900")),
						g.Text(otherUserName),
					),
					unreadBadge,
				),
				Div(
					g.If(conv.IsUnread, Class("text-sm font-bold text-gray-700 truncate")),
					g.If(!conv.IsUnread, Class("text-sm text-gray-700 truncate")),
					g.Text(conv.AdTitle),
				),
				Div(
					g.If(conv.IsUnread, Class("text-sm font-bold text-gray-400 text-right")),
					g.If(!conv.IsUnread, Class("text-sm text-gray-400 text-right")),
					g.Text(timeStr),
				),
			),
		),
	)
}

// MessagesList renders the list of messages in a conversation
func MessagesList(messages []messaging.Message, currentUserID int) g.Node {
	var messageNodes []g.Node
	for _, msg := range messages {
		messageNodes = append(messageNodes, MessageItem(msg, currentUserID))
	}

	return Div(
		Class("flex-1 overflow-y-auto p-4 space-y-4 min-h-0"),
		g.If(len(messages) == 0,
			Div(
				Class("text-center py-8"),
				P(Class("text-gray-500"), g.Text("No messages yet. Start the conversation!")),
			),
		),
		g.Group(messageNodes),
	)
}

// MessageItem renders a single message
func MessageItem(msg messaging.Message, currentUserID int) g.Node {
	isOwnMessage := msg.SenderID == currentUserID

	messageClass := "bg-blue-500 text-white"
	containerClass := "flex justify-end"
	timeClass := "text-right text-xs text-gray-400 mr-2"
	if !isOwnMessage {
		messageClass = "bg-gray-200 text-gray-800"
		containerClass = "flex justify-start"
		timeClass = "text-left text-xs text-gray-400 ml-2"
	}

	return Div(
		Class(containerClass),
		Div(
			Class("flex items-end gap-2"),
			g.If(!isOwnMessage,
				Div(
					Class(timeClass),
					g.Text(formatAdAge(msg.CreatedAt)),
				),
			),
			Div(
				Class(fmt.Sprintf("max-w-xs lg:max-w-md px-4 py-2 rounded-lg %s", messageClass)),
				P(Class("text-sm"), g.Text(msg.Content)),
			),
			g.If(isOwnMessage,
				Div(
					Class(timeClass),
					g.Text(formatAdAge(msg.CreatedAt)),
				),
			),
		),
	)
}

// MessageForm renders the form for sending new messages
func MessageForm(conversationID int, targetID string) g.Node {
	return Form(
		Class("flex gap-3 p-4 bg-white border-t"),
		hx.Post(fmt.Sprintf("/messages/%d/send", conversationID)),
		hx.Target(targetID),
		hx.Swap("outerHTML"),
		Input(
			Type("text"),
			Name("message"),
			Placeholder("Type your message..."),
			Class("flex-1 p-3 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"),
			Required(),
		),
		Button(
			Type("submit"),
			Class("px-6 py-3 bg-blue-500 text-white font-medium rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"),
			g.Text("Send"),
		),
	)
}

// ConversationEditor is a unified component for editing conversations
// It can be used both inline within ads and in the messages page
func ConversationEditor(conv messaging.Conversation, messages []messaging.Message, currentUser user.User, isInline bool, htmxTarget, view string) g.Node {
	// Get the other user's name
	var otherUserName string
	if conv.User1ID == currentUser.ID {
		otherUserName = conv.User2Name
	} else {
		otherUserName = conv.User1Name
	}

	// Format conversation age
	ageStr := formatAdAge(conv.CreatedAt)

	// Determine close button behavior based on context
	var closeButton g.Node
	if isInline {
		// Inline mode: return to expanded ad (the same view that was there before)
		closeButton = Button(
			Type("button"),
			Class("absolute -top-2 -right-2 bg-gray-800 bg-opacity-80 text-white text-2xl font-bold rounded-full w-10 h-10 flex items-center justify-center shadow-lg z-30 hover:bg-gray-700 focus:outline-none"),
			hx.Get(fmt.Sprintf("/ad/detail/%d?view=%s", conv.AdID, view)),
			hx.Target(htmxTarget),
			hx.Swap("outerHTML"),
			g.Text("×"),
		)
	} else {
		// Messages page mode: collapse conversation
		closeButton = Button(
			Type("button"),
			Class("absolute -top-2 -right-2 bg-gray-800 bg-opacity-80 text-white text-2xl font-bold rounded-full w-10 h-10 flex items-center justify-center shadow-lg z-30 hover:bg-gray-700 focus:outline-none"),
			hx.Get(fmt.Sprintf("/messages/%d/collapse", conv.ID)),
			hx.Target(fmt.Sprintf("#conversation-%d", conv.ID)),
			hx.Swap("outerHTML"),
			g.Text("×"),
		)
	}

	// Determine the container ID based on context
	containerID := fmt.Sprintf("conversation-%d", conv.ID)
	if isInline {
		containerID = fmt.Sprintf("ad-%d", conv.AdID)
	}

	return Div(
		ID(containerID),
		Class("border rounded-lg shadow-lg bg-white flex flex-col relative"),
		closeButton,
		// Conversation header
		Div(
			Class("p-4 border-b bg-gray-50"),
			Div(
				Class("flex flex-col gap-1"),
				Div(Class("text-sm text-gray-600"), g.Text("To: "+otherUserName)),
				Div(Class("text-sm text-gray-600"), g.Text("Subject: Re: "+conv.AdTitle)),
				Div(Class("text-xs text-gray-400"), g.Text("Conversation started "+ageStr)),
			),
		),
		// Messages area
		Div(
			Class("flex-1 p-4 overflow-y-auto max-h-96"),
			ID(fmt.Sprintf("messages-%d", conv.ID)),
			MessagesList(messages, currentUser.ID),
		),
		// Message form
		Div(Class("p-4 border-t bg-gray-50"), MessageForm(conv.ID, "#"+containerID)),
	)
}

// ConversationInline renders a conversation inline within an ad view
func ConversationInline(conv messaging.Conversation, messages []messaging.Message, adObj ad.Ad, adOwner user.User, currentUser user.User, htmxTarget, view string) g.Node {
	// Use the unified conversation editor
	return ConversationEditor(conv, messages, currentUser, true, htmxTarget, view)
}

// ExpandedConversation renders an expanded conversation in the messages page
func ExpandedConversation(currentUser user.User, conv messaging.Conversation, messages []messaging.Message) g.Node {
	// Use the unified conversation editor
	return ConversationEditor(conv, messages, currentUser, false, "", "")
}
