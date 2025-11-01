package ui

import (
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// ---- Form Components ----

func formContainer(formID string, content ...g.Node) g.Node {
	return Form(
		ID(formID),
		Class("space-y-6"),
		g.Group(content),
	)
}

func formGroup(labelText string, fieldID string, input g.Node) g.Node {
	return Div(
		Class("space-y-2"),
		Label(For(fieldID), Class("block"), g.Text(labelText)),
		input,
	)
}

func Checkbox(id string, value string, label string, checked bool, disabled bool, attrs ...g.Node) g.Node {
	inputAttrs := []g.Node{
		Type("checkbox"),
		Name(id),
		Value(value),
		ID(id + "-" + value),
	}

	if checked {
		inputAttrs = append(inputAttrs, Checked())
	}
	if disabled {
		inputAttrs = append(inputAttrs, Disabled())
		inputAttrs = append(inputAttrs, g.Attr("class", "opacity-50 cursor-not-allowed"))
	}

	inputAttrs = append(inputAttrs, attrs...)

	labelNode := Label(
		For(id+"-"+value),
		func() g.Node {
			if disabled {
				return Class("text-gray-400")
			}
			return g.Text("")
		}(),
		g.Text(label),
	)

	return Div(
		Class("flex items-center space-x-2"),
		Input(inputAttrs...),
		labelNode,
	)
}

func TextInput(id, name, value string) g.Node {
	return Input(
		Type("text"),
		ID(id),
		Name(name),
		Value(value),
		Class("w-full p-2 border rounded"),
	)
}

func passwordInput(id, name string) g.Node {
	return Input(
		Type("password"),
		ID(id),
		Name(name),
		Class("w-full p-2 border rounded"),
	)
}

// NotificationMethodRadioGroup returns the notification method radio group UI component
func NotificationMethodRadioGroup(selectedMethod string, emailAddress *string, phoneNumber string, smsOptedOut bool) g.Node {
	radioButtons := []g.Node{
		Div(Class("flex items-start flex-col"),
			Div(Class("flex items-center"),
				Input(Type("radio"), Name("notificationMethod"), Value("sms"), ID("notificationMethod-sms"), g.If(selectedMethod == "sms", Checked()), Class("mr-2"),
					hx.Post("/api/notification-method-changed"),
					hx.Target("#emailField"),
					hx.Swap("innerHTML"),
					hx.Include("this"),
				),
				Label(For("notificationMethod-sms"), g.Text("Text to "+phoneNumber)),
			),
			g.If(smsOptedOut,
				Div(ID("unstopControl"), Class("ml-6 mt-2 text-sm"),
					Div(Class("text-red-600 mb-2"), g.Text("SMS is currently paused (STOP).")),
					Button(Type("button"), Class("px-3 py-1 bg-green-600 text-white rounded"), g.Text("Resume SMS"),
						hx.Post("/api/unstop-sms"),
						hx.Target("#notificationMethodGroup"),
						hx.Swap("innerHTML"),
					),
				),
			),
		),
		Div(Class("flex items-center"),
			Input(Type("radio"), Name("notificationMethod"), Value("email"), ID("notificationMethod-email"), g.If(selectedMethod == "email", Checked()), Class("mr-2"),
				hx.Post("/api/notification-method-changed"),
				hx.Target("#emailField"),
				hx.Swap("innerHTML"),
				hx.Include("this"),
			),
			Label(For("notificationMethod-email"), g.Text("Email to:")),
		),
		Div(ID("emailField"), Class("ml-6 mt-2"),
			g.If(selectedMethod == "email",
				Input(
					Type("text"),
					ID("emailAddress"),
					Name("emailAddress"),
					Placeholder("Enter email address"),
					Value(func() string {
						if emailAddress != nil {
							return *emailAddress
						}
						return ""
					}()),
					Class("w-full p-2 border rounded"),
					Required(),
				),
			),
			g.If(selectedMethod != "email",
				Input(
					Type("text"),
					ID("emailAddress"),
					Name("emailAddress"),
					Placeholder("Enter email address"),
					Value(func() string {
						if emailAddress != nil {
							return *emailAddress
						}
						return ""
					}()),
					Class("w-full p-2 border rounded opacity-50 cursor-not-allowed"),
					Disabled(),
				),
			),
		),
		Div(Class("flex items-center"),
			Input(Type("radio"), Name("notificationMethod"), Value("signal"), ID("notificationMethod-signal"), g.If(selectedMethod == "signal", Checked()), Class("mr-2"),
				hx.Post("/api/notification-method-changed"),
				hx.Target("#emailField"),
				hx.Swap("innerHTML"),
				hx.Include("this"),
			),
			Label(For("notificationMethod-signal"), g.Text("Signal")),
		),
	}

	return formGroup("Notification Method", "notificationMethod",
		Div(Class("space-y-3"),
			g.Group(radioButtons),
		),
	)
}
