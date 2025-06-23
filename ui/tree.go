package ui

import (
	"fmt"
	"net/url"

	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// TreeNode represents a node in the tree view.
func TreeNode(name, path string, level int) g.Node {
	return Div(
		Style(fmt.Sprintf("padding-left: %dem;", level*2)), // Increased padding
		Button(
			Class("btn btn-xs btn-outline"),
			hx.Get(fmt.Sprintf("/tree%v", path)),
			hx.Target("next .children"),
			hx.Swap("outerHTML"), // Swap the button and the children div
			g.Text("+"),
		),
		Span(Class("ml-2"), g.Text(name)),
		Div(Class("children")),
	)
}

func CollapsedTreeNode(name, path, q string, level int) g.Node {
	return Div(
		Class("ml-4"),
		Button(
			Class("btn btn-xs btn-ghost"),
			hx.Get(fmt.Sprintf("/tree%s?q=%s", path, url.QueryEscape(q))),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("+"),
		),
		g.Text(name),
	)
}

func ExpandedTreeNode(name, path, q string, level int, children g.Node) g.Node {
	return Div(
		Class("ml-4"),
		Button(
			Class("btn btn-xs btn-ghost"),
			hx.Get(fmt.Sprintf("/tree-collapsed%s?q=%s", path, url.QueryEscape(q))),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("-"),
		),
		g.Text(name),
		children,
	)
}
