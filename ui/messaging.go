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

// ConversationsList renders the list of conversations
func ConversationsList(conversations []messaging.Conversation, currentUserID int) g.Node {
	var conversationNodes []g.Node
	for _, conv := range conversations {
		conversationNodes = append(conversationNodes, ConversationListItem(conv, currentUserID))
	}

	return Div(
		Class("space-y-4"),
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

	// Format the last message time
	timeStr := "No messages yet"
	if !conv.LastMessageAt.IsZero() {
		timeStr = formatAdAge(conv.LastMessageAt)
	}

	// Show unread count if any
	unreadBadge := g.Node(nil)
	if conv.UnreadCount > 0 {
		unreadBadge = Span(
			Class("bg-blue-500 text-white text-xs font-bold px-2 py-1 rounded-full"),
			g.Text(fmt.Sprintf("%d", conv.UnreadCount)),
		)
	}

	return A(
		Href(fmt.Sprintf("/messages/%d", conv.ID)),
		Class("block border rounded-lg p-4 hover:bg-gray-50 transition-colors"),
		Div(
			Class("flex items-center justify-between"),
			Div(
				Class("flex-1"),
				Div(
					Class("flex items-center gap-2"),
					H3(Class("font-semibold text-lg"), g.Text(otherUserName)),
					unreadBadge,
				),
				P(Class("text-gray-600 text-sm"), g.Text(conv.AdTitle)),
				g.If(conv.LastMessage != "",
					P(Class("text-gray-500 text-sm truncate"), g.Text(conv.LastMessage)),
				),
			),
			Div(
				Class("text-right text-sm text-gray-400"),
				g.Text(timeStr),
			),
		),
	)
}

// ConversationPage renders a specific conversation page
func ConversationPage(currentUser *user.User, conversation messaging.Conversation, messages []messaging.Message) g.Node {
	// Determine the other participant's name
	otherUserName := conversation.User1Name
	if currentUser.ID == conversation.User1ID {
		otherUserName = conversation.User2Name
	}

	return Page(
		fmt.Sprintf("Conversation with %s", otherUserName),
		currentUser,
		fmt.Sprintf("/messages/%d", conversation.ID),
		[]g.Node{
			Div(
				Class("flex items-center gap-4 mb-6"),
				A(
					Href("/messages"),
					Class("text-blue-500 hover:underline"),
					g.Text("← Back to messages"),
				),
				PageHeader(fmt.Sprintf("Conversation with %s", otherUserName)),
			),
			ContentContainer(
				Div(
					Class("mb-4 p-3 bg-gray-50 rounded-lg"),
					P(Class("text-sm text-gray-600"), g.Text(fmt.Sprintf("About: %s", conversation.AdTitle))),
				),
				MessagesList(messages, currentUser.ID),
				MessageForm(conversation.ID),
			),
		},
	)
}

// MessagesList renders the list of messages in a conversation
func MessagesList(messages []messaging.Message, currentUserID int) g.Node {
	var messageNodes []g.Node
	for _, msg := range messages {
		messageNodes = append(messageNodes, MessageItem(msg, currentUserID))
	}

	return Div(
		Class("space-y-4 mb-6"),
		g.Group(messageNodes),
	)
}

// MessageItem renders a single message
func MessageItem(msg messaging.Message, currentUserID int) g.Node {
	isOwnMessage := msg.SenderID == currentUserID

	messageClass := "bg-blue-500 text-white"
	containerClass := "flex justify-end"
	if !isOwnMessage {
		messageClass = "bg-gray-200 text-gray-800"
		containerClass = "flex justify-start"
	}

	return Div(
		Class(containerClass),
		Div(
			Class(fmt.Sprintf("max-w-xs lg:max-w-md px-4 py-2 rounded-lg %s", messageClass)),
			P(Class("text-sm"), g.Text(msg.Content)),
			Div(
				Class("text-xs opacity-75 mt-1"),
				g.Text(formatAdAge(msg.CreatedAt)),
			),
		),
	)
}

// MessageForm renders the form for sending new messages
func MessageForm(conversationID int) g.Node {
	return Form(
		Class("flex gap-2"),
		hx.Post(fmt.Sprintf("/messages/%d/send", conversationID)),
		hx.Target("body"),
		hx.Swap("outerHTML"),
		Input(
			Type("text"),
			Name("message"),
			Placeholder("Type your message..."),
			Class("flex-1 p-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"),
			Required(),
		),
		Button(
			Type("submit"),
			Class("px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-500"),
			g.Text("Send"),
		),
	)
}
