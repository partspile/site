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
			Class("border rounded px-1 py-0.5 text-xs"),
			hx.Get(fmt.Sprintf("/tree%v", path)),
			hx.Target("next .children"),
			hx.Swap("outerHTML"), // Swap the button and the children div
			g.Text("+"),
		),
		Span(Class("ml-2"), g.Text(name)),
		Div(Class("children")),
	)
}

func CollapsedTreeNode(name, path, q, structuredQuery string, level int) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5 text-xs"),
			hx.Get(fmt.Sprintf("/tree%s?q=%s&structured_query=%s", path, url.QueryEscape(q), url.QueryEscape(structuredQuery))),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("+"),
		),
		g.Text(decodedName),
	)
}

func CollapsedTreeNodeWithThreshold(name, path, q, structuredQuery, threshold string, level int) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5 text-xs"),
			hx.Get(fmt.Sprintf("/tree%s?q=%s&structured_query=%s&threshold=%s", path, url.QueryEscape(q), url.QueryEscape(structuredQuery), threshold)),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("+"),
		),
		g.Text(decodedName),
	)
}

func ExpandedTreeNode(name, path, q, structuredQuery string, level int, children g.Node) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5 text-xs"),
			hx.Get(fmt.Sprintf("/tree-collapsed%s?q=%s&structured_query=%s", path, url.QueryEscape(q), url.QueryEscape(structuredQuery))),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("-"),
		),
		g.Text(decodedName),
		children,
	)
}

func ExpandedTreeNodeWithThreshold(name, path, q, structuredQuery, threshold string, level int, children g.Node) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5 text-xs"),
			hx.Get(fmt.Sprintf("/tree-collapsed%s?q=%s&structured_query=%s&threshold=%s", path, url.QueryEscape(q), url.QueryEscape(structuredQuery), threshold)),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("-"),
		),
		g.Text(decodedName),
		children,
	)
}
