package ui

import (
	"fmt"
	"strings"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vehicle"
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
				Li(A(Href("/admin/makes"), g.Text("Manage Makes"))),
				Li(A(Href("/admin/models"), g.Text("Manage Models"))),
				Li(A(Href("/admin/years"), g.Text("Manage Years"))),
				Li(A(Href("/admin/part-categories"), g.Text("Manage Part Categories"))),
				Li(A(Href("/admin/part-sub-categories"), g.Text("Manage Part Sub-Categories"))),
			),
		},
	)
}

func AdminUsers(currentUser *user.User, path string, users []user.User, status string) g.Node {
	return Page(
		"Admin - Users",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("User Management")),
			Div(
				Class("flex justify-between items-center my-4"),
				AdminStatusSelector("users", status),
				A(
					Class("px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"),
					Href(fmt.Sprintf("/api/admin/export/users?status=%s", status)),
					g.Text("Export Users"),
				),
			),
			AdminUserTable(users, status),
		},
	)
}

func AdminUserTable(users []user.User, status string) g.Node {
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
							func() g.Node {
								if status == "dead" {
									return Button(
										g.Text("Resurrect"),
										Class("px-2 py-1 bg-green-500 text-white rounded hover:bg-green-600"),
										hx.Post(fmt.Sprintf("/api/admin/users/resurrect/%d", u.ID)),
										hx.Target("#adminUserTable"),
										hx.Swap("outerHTML"),
									)
								}
								return Div(
									Class("flex space-x-2"),
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
									Button(
										g.Text("Kill"),
										Class("px-2 py-1 bg-red-500 text-white rounded hover:bg-red-600"),
										hx.Delete(fmt.Sprintf("/api/admin/users/kill/%d", u.ID)),
										hx.Target("#adminUserTable"),
										hx.Swap("outerHTML"),
										hx.Confirm("Are you sure you want to kill this user? This will also remove all of their ads."),
									),
								)
							}(),
						),
					)
				})),
			),
		),
	)
}

func AdminAds(currentUser *user.User, path string, ads []ad.Ad, status string) g.Node {
	return Page(
		"Admin - Ads",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Ad Management")),
			Div(
				Class("flex justify-between items-center my-4"),
				AdminStatusSelector("ads", status),
				A(
					Class("px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"),
					Href(fmt.Sprintf("/api/admin/export/ads?status=%s", status)),
					g.Text("Export Ads"),
				),
			),
			AdminAdTable(ads, status),
		},
	)
}

func AdminAdTable(ads []ad.Ad, status string) g.Node {
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
							func() g.Node {
								if status == "dead" {
									return Button(
										g.Text("Resurrect"),
										Class("px-2 py-1 bg-green-500 text-white rounded hover:bg-green-600"),
										hx.Post(fmt.Sprintf("/api/admin/ads/resurrect/%d", a.ID)),
										hx.Target("#adminAdTable"),
										hx.Swap("outerHTML"),
									)
								}
								return Div(
									Class("flex space-x-2"),
									StyledLink("View", fmt.Sprintf("/ad/%d", a.ID), ButtonPrimary),
									Button(
										g.Text("Kill"),
										Class("px-2 py-1 bg-red-500 text-white rounded hover:bg-red-600"),
										hx.Delete(fmt.Sprintf("/api/admin/ads/kill/%d", a.ID)),
										hx.Target("#adminAdTable"),
										hx.Swap("outerHTML"),
										hx.Confirm("Are you sure you want to kill this ad?"),
									),
								)
							}(),
						),
					)
				})),
			),
		),
	)
}

func AdminStatusSelector(page, currentStatus string) g.Node {
	return Div(
		Class("my-4"),
		A(
			Href(fmt.Sprintf("/admin/%s", page)),
			g.Text("Active"),
			func() g.Node {
				if currentStatus != "dead" {
					return Class("px-4 py-2 bg-blue-500 text-white rounded-l")
				}
				return Class("px-4 py-2 bg-gray-300 hover:bg-gray-400 text-gray-800 rounded-l")
			}(),
		),
		A(
			Href(fmt.Sprintf("/admin/%s?status=dead", page)),
			g.Text("Dead"),
			func() g.Node {
				if currentStatus == "dead" {
					return Class("px-4 py-2 bg-blue-500 text-white rounded-r")
				}
				return Class("px-4 py-2 bg-gray-300 hover:bg-gray-400 text-gray-800 rounded-r")
			}(),
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
			Div(
				Class("my-4"),
				A(
					Class("px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"),
					Href("/api/admin/export/transactions"),
					g.Text("Export Transactions"),
				),
			),
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

func AdminMakes(currentUser *user.User, path string, makes []vehicle.Make) g.Node {
	return Page(
		"Admin - Makes",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Make Management")),
			Table(
				Class("min-w-full border border-gray-300 bg-white shadow-sm"),
				THead(
					Tr(Class("bg-gray-200"),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("ID")),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Make")),
					),
				),
				TBody(
					g.Group(g.Map(makes, func(m vehicle.Make) g.Node {
						return Tr(Class("hover:bg-gray-50"),
							Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", m.ID)),
							Td(Class("border border-gray-300 px-4 py-2"), g.Text(m.Name)),
						)
					})),
				),
			),
		},
	)
}

func AdminModels(currentUser *user.User, path string, models []vehicle.Model) g.Node {
	return Page(
		"Admin - Models",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Model Management")),
			Table(
				Class("min-w-full border border-gray-300 bg-white shadow-sm"),
				THead(
					Tr(Class("bg-gray-200"),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("ID")),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Model")),
					),
				),
				TBody(
					g.Group(g.Map(models, func(m vehicle.Model) g.Node {
						return Tr(Class("hover:bg-gray-50"),
							Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", m.ID)),
							Td(Class("border border-gray-300 px-4 py-2"), g.Text(m.Name)),
						)
					})),
				),
			),
		},
	)
}

func AdminYears(currentUser *user.User, path string, years []vehicle.Year) g.Node {
	return Page(
		"Admin - Years",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Year Management")),
			Table(
				Class("min-w-full border border-gray-300 bg-white shadow-sm"),
				THead(
					Tr(Class("bg-gray-200"),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("ID")),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Year")),
					),
				),
				TBody(
					g.Group(g.Map(years, func(y vehicle.Year) g.Node {
						return Tr(Class("hover:bg-gray-50"),
							Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", y.ID)),
							Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", y.Year)),
						)
					})),
				),
			),
		},
	)
}

func AdminPartCategories(currentUser *user.User, path string, categories []part.Category) g.Node {
	return Page(
		"Admin - Part Categories",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Part Category Management")),
			Table(
				Class("min-w-full border border-gray-300 bg-white shadow-sm"),
				THead(
					Tr(Class("bg-gray-200"),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("ID")),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Name")),
					),
				),
				TBody(
					g.Group(g.Map(categories, func(c part.Category) g.Node {
						return Tr(Class("hover:bg-gray-50"),
							Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", c.ID)),
							Td(Class("border border-gray-300 px-4 py-2"), g.Text(c.Name)),
						)
					})),
				),
			),
		},
	)
}

func AdminPartSubCategories(currentUser *user.User, path string, subCategories []part.SubCategory) g.Node {
	return Page(
		"Admin - Part Sub-Categories",
		currentUser,
		path,
		[]g.Node{
			H1(g.Text("Part Sub-Category Management")),
			Table(
				Class("min-w-full border border-gray-300 bg-white shadow-sm"),
				THead(
					Tr(Class("bg-gray-200"),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("ID")),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Category ID")),
						Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Name")),
					),
				),
				TBody(
					g.Group(g.Map(subCategories, func(sc part.SubCategory) g.Node {
						return Tr(Class("hover:bg-gray-50"),
							Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", sc.ID)),
							Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", sc.CategoryID)),
							Td(Class("border border-gray-300 px-4 py-2"), g.Text(sc.Name)),
						)
					})),
				),
			),
		},
	)
}
