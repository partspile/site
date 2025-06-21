package ui

import (
	"fmt"
	"strings"

	"github.com/parts-pile/site/ad"
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

func AdminAds(currentUser *user.User, path string, ads []ad.Ad) g.Node {
	return Page(
		"Admin - Ads",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Ad Management")),
			AdminAdTable(ads),
		},
	)
}

func AdminAdTable(ads []ad.Ad) g.Node {
	return Div(
		ID("adminAdTable"),
		Class("overflow-x-auto"),
		Table(
			Class("min-w-full border border-gray-300 bg-white shadow-sm"),
			THead(
				Tr(
					Class("bg-gray-200"),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("ID")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Make")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Years")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Models")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Price")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Actions")),
				),
			),
			TBody(
				g.Group(g.Map(ads, func(a ad.Ad) g.Node {
					return Tr(
						Class("hover:bg-gray-50"),
						Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", a.ID)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(a.Make)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(strings.Join(a.Years, ", "))),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(strings.Join(a.Models, ", "))),
						Td(Class("border border-gray-300 px-4 py-2"), g.Textf("$%.2f", a.Price)),
						Td(Class("border border-gray-300 px-4 py-2"),
							A(Href(fmt.Sprintf("/ad/%d", a.ID)), g.Text("View")),
						),
					)
				})),
			),
		),
	)
}

func AdminTransactions(currentUser *user.User, path string, transactions []user.Transaction) g.Node {
	return Page(
		"Admin - Transactions",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Transaction Log")),
			AdminTransactionTable(transactions),
		},
	)
}

func AdminTransactionTable(transactions []user.Transaction) g.Node {
	return Div(
		ID("adminTransactionTable"),
		Class("overflow-x-auto"),
		Table(
			Class("min-w-full border border-gray-300 bg-white shadow-sm"),
			THead(
				Tr(
					Class("bg-gray-200"),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("ID")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("User ID")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Amount")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Type")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Date")),
				),
			),
			TBody(
				g.Group(g.Map(transactions, func(t user.Transaction) g.Node {
					return Tr(
						Class("hover:bg-gray-50"),
						Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", t.ID)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", t.UserID)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Textf("$%.2f", t.Amount)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(t.Type)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(t.CreatedAt.Format("2006-01-02 15:04:05"))),
					)
				})),
			),
		),
	)
}

func AdminExport(currentUser *user.User, path string) g.Node {
	return Page(
		"Admin - Export",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Export Data")),
			P(g.Text("Select the data you would like to export as a CSV file.")),
			Div(
				Class("flex space-x-4"),
				A(
					Class("px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"),
					Href("/api/admin/export/users"),
					g.Text("Export Users"),
				),
				A(
					Class("px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"),
					Href("/api/admin/export/ads"),
					g.Text("Export Ads"),
				),
				A(
					Class("px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"),
					Href("/api/admin/export/transactions"),
					g.Text("Export Transactions"),
				),
			),
		},
	)
}
