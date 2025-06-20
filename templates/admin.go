package templates

import (
	"fmt"

	"github.com/parts-pile/site/user"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func AdminDashboard(currentUser *user.User, path string) g.Node {
	return Page(
		"Admin Dashboard",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Admin Dashboard")),
			P(g.Text("Welcome to the admin dashboard.")),
			Ul(
				Li(A(Href("/admin/users"), g.Text("Manage Users"))),
				Li(A(Href("/admin/ads"), g.Text("Manage Ads"))),
				Li(A(Href("/admin/transactions"), g.Text("View Transactions"))),
				Li(A(Href("/admin/export"), g.Text("Export Data"))),
			),
		},
	)
}

func AdminUsers(currentUser *user.User, path string, users []user.User) g.Node {
	return Page(
		"Admin - Users",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("User Management")),
			AdminUserTable(users),
		},
	)
}

func AdminUserTable(users []user.User) g.Node {
	return Div(
		ID("adminUserTable"),
		Class("overflow-x-auto"),
		Table(
			Class("min-w-full border border-gray-300 bg-white shadow-sm"),
			THead(
				Tr(
					Class("bg-gray-200"),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("ID")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Name")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Phone")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Is Admin")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Actions")),
				),
			),
			TBody(
				g.Group(g.Map(users, func(u user.User) g.Node {
					return Tr(
						Class("hover:bg-gray-50"),
						Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", u.ID)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(u.Name)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(u.Phone)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%v", u.IsAdmin)),
						Td(Class("border border-gray-300 px-4 py-2"),
							Form(
								hx.Post("/api/admin/users/set-admin"),
								hx.Target("#adminUserTable"),
								hx.Swap("outerHTML"),
								Input(Type("hidden"), Name("user_id"), Value(fmt.Sprintf("%d", u.ID))),
								Input(Type("hidden"), Name("is_admin"), Value(fmt.Sprintf("%v", !u.IsAdmin))),
								Button(
									Type("submit"),
									Class("px-2 py-1 bg-blue-500 text-white rounded hover:bg-blue-600"),
									g.Text("Toggle Admin"),
								),
							),
						),
					)
				})),
			),
		),
	)
}
