package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"

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
			pageHeader("Messages"),
			contentContainer(
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
			pageHeader("Messages"),
			contentContainer(
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
			conversationNodes = append(conversationNodes, ExpandedConversation(currentUser, conv, expandedMessages))
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

// ExpandedConversation renders an expanded conversation view within the list
func ExpandedConversation(currentUser *user.User, conversation messaging.Conversation, messages []messaging.Message) g.Node {
	// Determine the other participant's name
	otherUserName := conversation.User1Name
	if currentUser.ID == conversation.User1ID {
		otherUserName = conversation.User2Name
	}

	return Div(
		ID(fmt.Sprintf("conversation-%d", conversation.ID)),
		Class("bg-gray-50"),
		Div(
			Class("p-4"),
			Div(
				Class("flex items-center justify-between mb-4"),
				Div(
					Class("space-y-1"),
					Div(
						Class("text-sm text-gray-600"),
						Span(Class("font-medium"), g.Text("To: ")),
						g.Text(otherUserName),
					),
					Div(
						Class("text-sm text-gray-600"),
						Span(Class("font-medium"), g.Text("Subject: ")),
						g.Text(fmt.Sprintf("Re: %s", conversation.AdTitle)),
					),
				),
				Button(
					Class("text-gray-400 hover:text-gray-600 p-1"),
					hx.Get(fmt.Sprintf("/messages/%d/collapse", conversation.ID)),
					hx.Target(fmt.Sprintf("#conversation-%d", conversation.ID)),
					hx.Swap("outerHTML"),
					Title("Close conversation"),
					g.Text("✕"),
				),
			),
			Div(
				Class("bg-white rounded-lg border h-96 flex flex-col"),
				MessagesList(messages, currentUser.ID),
				MessageForm(conversation.ID),
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
func MessageForm(conversationID int) g.Node {
	return Form(
		Class("flex gap-3 p-4 bg-white border-t"),
		hx.Post(fmt.Sprintf("/messages/%d/send", conversationID)),
		hx.Target(fmt.Sprintf("#conversation-%d", conversationID)),
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
			Class("px-6 py-3 bg-blue-500 text-white font-medium rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-colors"),
			g.Text("Send"),
		),
	)
}
