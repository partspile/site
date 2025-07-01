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

// AdminSectionPage renders the admin section navigation and the current section content.
func AdminSectionPage(currentUser *user.User, path, activeSection string, content g.Node) g.Node {
	sections := []struct {
		Name  string
		Label string
	}{
		{"users", "Users"},
		{"ads", "Ads"},
		{"transactions", "Transactions"},
		{"makes", "Makes"},
		{"models", "Models"},
		{"years", "Years"},
		{"part-categories", "Part Categories"},
		{"part-sub-categories", "Part Sub-Categories"},
		{"parent-companies", "Parent Companies"},
		{"make-parent-companies", "Make-Parent Companies"},
	}
	return Div(
		ID("admin-section"),
		Class("my-8"),
		H1(g.Text("Admin Dashboard")),
		Div(
			Class("flex flex-wrap gap-2 mb-6"),
			g.Group(g.Map(sections, func(s struct{ Name, Label string }) g.Node {
				colorClass := "bg-gray-200 text-gray-800 hover:bg-gray-300"
				if s.Name == activeSection {
					colorClass = "bg-blue-500 text-white"
				}
				return Button(
					Class("px-4 py-1 rounded "+colorClass),
					ID("btn-"+s.Name),
					hx.Get("/admin/"+s.Name),
					hx.Target("#admin-section"),
					hx.Swap("outerHTML"),
					g.Text(s.Label),
				)
			})),
		),
		Div(
			ID("admin-section-content"),
			content,
		),
	)
}

// Update AdminDashboard to default to users section
func AdminDashboard(currentUser *user.User, path string) g.Node {
	return Page(
		"Admin Dashboard",
		currentUser,
		path,
		[]g.Node{
			AdminSectionPage(currentUser, path, "users", nil),
		},
	)
}

// Update section UIs to return only the section content
func AdminUsersSection(users []user.User, status string) g.Node {
	return Div(
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
	)
}

func AdminAdsSection(ads []ad.Ad, status string) g.Node {
	return Div(
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
	)
}

func AdminUsers(currentUser *user.User, path string, users []user.User, status string) g.Node {
	return Page(
		"Admin - Users",
		currentUser,
		path,
		[]g.Node{
			AdminUsersSection(users, status),
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
								if status == "archived" {
									return Button(
										g.Text("Restore"),
										Class("px-2 py-1 bg-green-500 text-white rounded hover:bg-green-600"),
										hx.Post(fmt.Sprintf("/api/admin/users/restore/%d", u.ID)),
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
										g.Text("Archive"),
										Class("px-2 py-1 bg-red-500 text-white rounded hover:bg-red-600"),
										hx.Delete(fmt.Sprintf("/api/admin/users/archive/%d", u.ID)),
										hx.Target("#adminUserTable"),
										hx.Swap("outerHTML"),
										hx.Confirm("Are you sure you want to archive this user? This will also remove all of their ads."),
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
			AdminAdsSection(ads, status),
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
								if status == "archived" {
									return Div(
										Class("flex space-x-2"),
										StyledLink("View", fmt.Sprintf("/ad/%d", a.ID), ButtonPrimary),
										Button(
											g.Text("Restore"),
											Class("px-2 py-1 bg-green-500 text-white rounded hover:bg-green-600"),
											hx.Post(fmt.Sprintf("/api/admin/ads/restore/%d", a.ID)),
											hx.Target("#adminAdTable"),
											hx.Swap("outerHTML"),
										),
									)
								}
								return Div(
									Class("flex space-x-2"),
									StyledLink("View", fmt.Sprintf("/ad/%d", a.ID), ButtonPrimary),
									Button(
										g.Text("Archive"),
										Class("px-2 py-1 bg-red-500 text-white rounded hover:bg-red-600"),
										hx.Delete(fmt.Sprintf("/api/admin/ads/archive/%d", a.ID)),
										hx.Target("#adminAdTable"),
										hx.Swap("outerHTML"),
										hx.Confirm("Are you sure you want to archive this ad?"),
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
				if currentStatus != "archived" {
					return Class("px-4 py-2 bg-blue-500 text-white rounded-l")
				}
				return Class("px-4 py-2 bg-gray-300 hover:bg-gray-400 text-gray-800 rounded-l")
			}(),
		),
		A(
			Href(fmt.Sprintf("/admin/%s?status=archived", page)),
			g.Text("Archived"),
			func() g.Node {
				if currentStatus == "archived" {
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
			AdminTransactionsSection(transactions),
		},
	)
}

func AdminTransactionsSection(transactions []user.Transaction) g.Node {
	return Div(
		H1(g.Text("Transaction Log")),
		Div(
			Class("flex justify-end my-4"),
			A(
				Class("px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600"),
				Href("/api/admin/export/transactions"),
				g.Text("Export Transactions"),
			),
		),
		AdminTransactionTable(transactions),
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
			AdminMakesSection(makes),
		},
	)
}

func AdminMakesSection(makes []vehicle.Make) g.Node {
	return Div(
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
	)
}

func AdminModels(currentUser *user.User, path string, models []vehicle.Model) g.Node {
	return Page(
		"Admin - Models",
		currentUser,
		path,
		[]g.Node{
			AdminModelsSection(models),
		},
	)
}

func AdminModelsSection(models []vehicle.Model) g.Node {
	return Div(
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
	)
}

func AdminYears(currentUser *user.User, path string, years []vehicle.Year) g.Node {
	return Page(
		"Admin - Years",
		currentUser,
		path,
		[]g.Node{
			AdminYearsSection(years),
		},
	)
}

func AdminYearsSection(years []vehicle.Year) g.Node {
	return Div(
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
	)
}

func AdminPartCategories(currentUser *user.User, path string, categories []part.Category) g.Node {
	return Page(
		"Admin - Part Categories",
		currentUser,
		path,
		[]g.Node{
			AdminPartCategoriesSection(categories),
		},
	)
}

func AdminPartCategoriesSection(categories []part.Category) g.Node {
	return Div(
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
	)
}

func AdminPartSubCategories(currentUser *user.User, path string, subCategories []part.SubCategory) g.Node {
	return Page(
		"Admin - Part Sub-Categories",
		currentUser,
		path,
		[]g.Node{
			AdminPartSubCategoriesSection(subCategories),
		},
	)
}

func AdminPartSubCategoriesSection(subCategories []part.SubCategory) g.Node {
	return Div(
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
	)
}

func AdminParentCompanies(currentUser *user.User, path string, pcs []vehicle.ParentCompany) g.Node {
	return Page(
		"Admin - Parent Companies",
		currentUser,
		path,
		[]g.Node{
			AdminParentCompaniesSection(pcs),
		},
	)
}

func AdminParentCompaniesSection(pcs []vehicle.ParentCompany) g.Node {
	return Div(
		H1(g.Text("Parent Company Management")),
		Table(
			Class("min-w-full border border-gray-300 bg-white shadow-sm"),
			THead(
				Tr(Class("bg-gray-200"),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("ID")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Name")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Country")),
				),
			),
			TBody(
				g.Group(g.Map(pcs, func(pc vehicle.ParentCompany) g.Node {
					return Tr(Class("hover:bg-gray-50"),
						Td(Class("border border-gray-300 px-4 py-2"), g.Textf("%d", pc.ID)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(pc.Name)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(pc.Country)),
					)
				})),
			),
		),
	)
}

func AdminMakeParentCompaniesSection(rows []struct{ Make, ParentCompany string }) g.Node {
	return Div(
		H1(g.Text("Make-Parent Company Relationships")),
		Table(
			Class("min-w-full border border-gray-300 bg-white shadow-sm"),
			THead(
				Tr(Class("bg-gray-200"),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Make")),
					Th(Class("border border-gray-300 px-4 py-2 text-left font-semibold"), g.Text("Parent Company")),
				),
			),
			TBody(
				g.Group(g.Map(rows, func(row struct{ Make, ParentCompany string }) g.Node {
					return Tr(Class("hover:bg-gray-50"),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(row.Make)),
						Td(Class("border border-gray-300 px-4 py-2"), g.Text(row.ParentCompany)),
					)
				})),
			),
		),
	)
}
