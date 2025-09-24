package ui

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/parts-pile/site/ad"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func collapsedTreeNode(mode, name, path string) g.Node {
	expandPath := fmt.Sprintf("/tree-%s-expand/%s", mode, path)
	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5"),
			hx.Get(expandPath),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.If(mode == "search", hx.Include("[name='adIDs']")),
			g.Text("+ "+name),
		),
	)
}

func expandedTreeNode(mode, name, path string, level int, loc *time.Location, userID int, children []string, ads []ad.Ad) g.Node {

	var childNodes []g.Node

	// Handle ads at leaf level (level 6)
	if level == 6 && len(ads) > 0 {
		childNodes = listNodes(ads, userID, loc)
	} else if level == 6 && len(ads) == 0 {
		childNodes = append(childNodes, NoSearchResultsMessage())
	} else {
		// Show children as collapsed tree nodes
		for _, child := range children {
			childPath := strings.Join([]string{path, child}, "/")
			childNodes = append(childNodes, collapsedTreeNode(mode, child, childPath))
		}
	}

	collapsePath := fmt.Sprintf("/tree-%s-collapse/%s", mode, path)

	return Div(
		Class("ml-4"),
		Button(
			Class("hover:bg-gray-200 rounded px-1 py-0.5"),
			hx.Get(collapsePath),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("- "+name),
		),
		g.Group(childNodes),
	)
}

func ExpandedTreeNodeBrowse(name, currentPath string, level int, loc *time.Location, userID int, children []string, ads []ad.Ad) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return expandedTreeNode("browse", decodedName, currentPath, level, loc, userID, children, ads)
}

func CollapsedTreeNodeBrowse(name string, path string) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return collapsedTreeNode("browse", decodedName, path)
}

func CollapsedTreeNodeSearch(name string, path string) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return collapsedTreeNode("search", decodedName, path)
}

func ExpandedTreeNodeSearch(name, currentPath string, level int, loc *time.Location, userID int, children []string, ads []ad.Ad) g.Node {
	decodedName, _ := url.QueryUnescape(name)
	return expandedTreeNode("search", decodedName, currentPath, level, loc, userID, children, ads)
}
