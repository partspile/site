package ui

import (
	"fmt"
	"net/url"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/user"
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

func CollapsedTreeNode(name, path, q string, level int) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5"),
			hx.Get(fmt.Sprintf("/tree%s?q=%s", path, url.QueryEscape(q))),
			hx.Target("this"),
			hx.Swap("outerHTML"),
			g.Text("+ "+decodedName),
		),
	)
}

func CollapsedTreeNodeWithThreshold(name, path, q, threshold string, level int) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5 text-xs"),
			hx.Get(fmt.Sprintf("/tree%s?q=%s&threshold=%s", path, url.QueryEscape(q), threshold)),
			hx.Target("this"),
			hx.Swap("outerHTML"),
			g.Text("+"),
		),
		g.Text(decodedName),
	)
}

func ExpandedTreeNode(name, path, q string, level int, children g.Node) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5 text-xs"),
			hx.Get(fmt.Sprintf("/tree-collapsed%s?q=%s", path, url.QueryEscape(q))),
			hx.Target("this"),
			hx.Swap("outerHTML"),
			g.Text("-"),
		),
		g.Text(decodedName),
		children,
	)
}

func ExpandedTreeNodeWithThreshold(name, path, q, threshold string, level int, children g.Node) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5 text-xs"),
			hx.Get(fmt.Sprintf("/tree-collapsed%s?q=%s&threshold=%s", path, url.QueryEscape(q), threshold)),
			hx.Target("this"),
			hx.Swap("outerHTML"),
			g.Text("-"),
		),
		g.Text(decodedName),
		children,
	)
}

func collapsedTreeNode(name, path string) g.Node {
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5"),
			hx.Get(path),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("+ "+name),
		),
	)
}

func CollapsedTreeNodeBrowse(name string, fullPath string) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	path := fmt.Sprintf("/tree-browse-expand/%s", fullPath)
	return collapsedTreeNode(decodedName, path)
}

func ExpandedTreeNodeBrowse(name string, level int, children []string, ads []ad.Ad, currentUser *user.User, timezone string, currentPath string) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	collapsePath := fmt.Sprintf("/tree-browse-collapse/%s", name)

	var childNodes []g.Node

	// Handle ads at leaf level (level 6)
	if level == 6 && len(ads) > 0 {
		loc, _ := time.LoadLocation(timezone)
		for _, ad := range ads {
			childNodes = append(childNodes, AdCardCompactTree(ad, loc, currentUser))
		}
	} else if level == 6 && len(ads) == 0 {
		childNodes = append(childNodes, NoSearchResultsMessage())
	} else {
		// Show children as collapsed tree nodes
		for _, child := range children {
			// Construct full path for child
			var childPath string
			if currentPath == "" {
				childPath = child
			} else {
				childPath = currentPath + "/" + child
			}
			childNodes = append(childNodes, CollapsedTreeNodeBrowse(child, childPath))
		}
	}

	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5"),
			hx.Get(collapsePath),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("- "+decodedName),
		),
		g.Group(childNodes),
	)
}

// Search mode tree nodes (adIDs passed via DOM storage)
func CollapsedTreeNodeSearch(name string, level int, fullPath string) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	path := fmt.Sprintf("/tree-search-expand/%s", fullPath)
	return collapsedTreeNode(decodedName, path)
}

func ExpandedTreeNodeSearch(name string, level int, children []string, ads []ad.Ad, currentUser *user.User, timezone string, currentPath string) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	collapsePath := fmt.Sprintf("/tree-search-collapse/%s", name)

	var childNodes []g.Node

	// Handle ads at leaf level (level 6)
	if level == 6 && len(ads) > 0 {
		loc, _ := time.LoadLocation(timezone)
		for _, ad := range ads {
			childNodes = append(childNodes, AdCardCompactTree(ad, loc, currentUser))
		}
	} else if level == 6 && len(ads) == 0 {
		childNodes = append(childNodes, NoSearchResultsMessage())
	} else {
		// Show children as collapsed tree nodes
		for _, child := range children {
			// Construct full path for child
			var childPath string
			if currentPath == "" {
				childPath = child
			} else {
				childPath = currentPath + "/" + child
			}
			childNodes = append(childNodes, CollapsedTreeNodeSearch(child, level+1, childPath))
		}
	}

	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5 text-xs"),
			hx.Get(collapsePath),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("- "+decodedName),
		),
		g.Group(childNodes),
	)
}
