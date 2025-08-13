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
					Class("flex flex-col bg-white rounded-lg border"),
					Div(
						Class("p-4 border-b bg-gray-50"),
						Div(
							Class("space-y-2 text-sm"),
							Div(
								Class("text-gray-600"),
								Span(Class("font-medium"), g.Text("To: ")),
								g.Text(otherUserName),
							),
							Div(
								Class("text-gray-600"),
								Span(Class("font-medium"), g.Text("Subject: ")),
								g.Text(fmt.Sprintf("Re: %s", conversation.AdTitle)),
							),
						),
					),
					Div(
						Class("flex-1 min-h-96 relative"),
						ID("chat-container"),
						MessagesList(messages, currentUser.ID),
						Div(
							Class("absolute bottom-0 left-0 right-0 h-2 cursor-ns-resize bg-gray-200 hover:bg-gray-300 transition-colors"),
							ID("resize-handle"),
							Title("Drag to resize chat area"),
						),
					),
					MessageForm(conversation.ID),
				),
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
		ID("messages-list"),
		Class("space-y-4"),
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
		Class("flex gap-3 message-form"),
		hx.Post(fmt.Sprintf("/messages/%d/send", conversationID)),
		hx.Target("#messages-list"),
		hx.Swap("outerHTML"),
		hx.On("htmx:afterRequest", "document.getElementById('messages-list').scrollTop = document.getElementById('messages-list').scrollHeight"),
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
