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
	collapsePath := fmt.Sprintf("/tree-browse-collapse/%s", currentPath)

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
	collapsePath := fmt.Sprintf("/tree-search-collapse/%s", currentPath)

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
