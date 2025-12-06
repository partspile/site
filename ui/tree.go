package ui

import (
	"fmt"
	"net/url"
	"time"

	"github.com/parts-pile/site/ad"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

func collapsedTreeNode(mode, name string, fieldName string, parentValues map[string]string) g.Node {
	// Build query parameters: field name + field value + all parent values
	params := url.Values{}
	params.Set("field", fieldName)
	params.Set(fieldName, name) // Include the field value itself
	for key, value := range parentValues {
		params.Set(key, value)
	}
	expandPath := fmt.Sprintf("/tree-%s-expand?%s", mode, params.Encode())

	return Div(
		Class("ml-4"),
		A(
			Href("#"),
			Class("text-gray-700 hover:text-gray-900 hover:bg-gray-100 px-1 py-0.5 rounded cursor-pointer"),
			hx.Get(expandPath),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.If(mode == "search", hx.Include("[name='adIDs']")),
			g.Text("+ "+name),
		),
	)
}

func expandedTreeNode(mode, fieldValue string, fieldName string, nextFieldName string, parentValues map[string]string, level int, loc *time.Location, userID int, children []string, ads []ad.Ad, csrfToken string) g.Node {

	var childNodes []g.Node

	// Check if this is the ads level (no next field means we're at the end)
	if nextFieldName == "" && len(ads) > 0 {
		childNodes = listNodes(ads, userID, loc, csrfToken)
	} else if nextFieldName == "" && len(ads) == 0 {
		childNodes = append(childNodes, NoSearchResultsMessage())
	} else {
		// Show children as collapsed tree nodes
		// Build parent values for children (current field becomes parent)
		childParentValues := make(map[string]string)
		for k, v := range parentValues {
			childParentValues[k] = v
		}
		childParentValues[fieldName] = fieldValue

		// Build child nodes with next field name
		for _, child := range children {
			childNodes = append(childNodes, collapsedTreeNode(mode, child, nextFieldName, childParentValues))
		}
	}

	// Build collapse path with query params (same as expand but for collapse)
	params := url.Values{}
	params.Set("field", fieldName)
	params.Set(fieldName, fieldValue) // Include the field value itself
	for key, value := range parentValues {
		params.Set(key, value)
	}
	collapsePath := fmt.Sprintf("/tree-%s-collapse?%s", mode, params.Encode())

	return Div(
		Class("ml-4"),
		A(
			Href("#"),
			Class("text-gray-700 hover:text-gray-900 hover:bg-gray-100 px-1 py-0.5 rounded cursor-pointer"),
			hx.Get(collapsePath),
			hx.Target("closest .ml-4"),
			hx.Swap("outerHTML"),
			g.Text("- "+fieldValue),
		),
		g.Group(childNodes),
	)
}

func ExpandedTreeNodeBrowse(fieldValue, fieldName, nextFieldName string, parentValues map[string]string, level int, loc *time.Location, userID int, userName string, children []string, ads []ad.Ad, csrfToken string) g.Node {
	decodedValue, _ := url.QueryUnescape(fieldValue)
	return expandedTreeNode("browse", decodedValue, fieldName, nextFieldName, parentValues, level, loc, userID, children, ads, csrfToken)
}

func CollapsedTreeNodeBrowse(fieldValue, fieldName string, parentValues map[string]string) g.Node {
	decodedValue, _ := url.QueryUnescape(fieldValue)
	return collapsedTreeNode("browse", decodedValue, fieldName, parentValues)
}

func CollapsedTreeNodeSearch(fieldValue, fieldName string, parentValues map[string]string) g.Node {
	decodedValue, _ := url.QueryUnescape(fieldValue)
	return collapsedTreeNode("search", decodedValue, fieldName, parentValues)
}

func ExpandedTreeNodeSearch(fieldValue, fieldName, nextFieldName string, parentValues map[string]string, level int, loc *time.Location, userID int, userName string, children []string, ads []ad.Ad, csrfToken string) g.Node {
	decodedValue, _ := url.QueryUnescape(fieldValue)
	return expandedTreeNode("search", decodedValue, fieldName, nextFieldName, parentValues, level, loc, userID, children, ads, csrfToken)
}

// ExpandedPathNode represents a node in an auto-expanded path
type ExpandedPathNode struct {
	FieldValue   string
	FieldName    string
	FieldIndex   int
	ParentValues map[string]string
}

// ExpandedTreeNodeBrowseAutoExpanded renders nested expanded nodes for auto-expanded path
// expandedPath contains the nodes that were auto-expanded (single-child nodes)
// The deepest node shows its children (multiple) or ads
func ExpandedTreeNodeBrowseAutoExpanded(expandedPath []ExpandedPathNode, fields []ad.Field, finalNextFieldName string, finalChildren []string, finalAds []ad.Ad, loc *time.Location, userID int, userName string, csrfToken string) g.Node {
	// Render nested expanded nodes, starting from the deepest
	// Build from the innermost (deepest) node outward
	var renderNode func(int) g.Node

	renderNode = func(pathIndex int) g.Node {
		node := expandedPath[pathIndex]
		decodedValue, _ := url.QueryUnescape(node.FieldValue)

		var childNodes []g.Node

		if pathIndex == len(expandedPath)-1 {
			// This is the deepest node - show final children or ads
			if finalNextFieldName == "" && len(finalAds) > 0 {
				childNodes = listNodes(finalAds, userID, loc, csrfToken)
			} else if finalNextFieldName == "" && len(finalAds) == 0 {
				childNodes = append(childNodes, NoSearchResultsMessage())
			} else {
				// Show final children as collapsed nodes
				childParentValues := make(map[string]string)
				for k, v := range node.ParentValues {
					childParentValues[k] = v
				}
				childParentValues[node.FieldName] = node.FieldValue

				for _, child := range finalChildren {
					childNodes = append(childNodes, collapsedTreeNode("browse", child, finalNextFieldName, childParentValues))
				}
			}
		} else {
			// This is an intermediate node - show the next node in the path as expanded
			childNodes = []g.Node{renderNode(pathIndex + 1)}
		}

		// Build collapse path
		params := url.Values{}
		params.Set("field", node.FieldName)
		params.Set(node.FieldName, node.FieldValue)
		for key, value := range node.ParentValues {
			params.Set(key, value)
		}
		collapsePath := fmt.Sprintf("/tree-browse-collapse?%s", params.Encode())

		return Div(
			Class("ml-4"),
			A(
				Href("#"),
				Class("text-gray-700 hover:text-gray-900 hover:bg-gray-100 px-1 py-0.5 rounded cursor-pointer"),
				hx.Get(collapsePath),
				hx.Target("closest .ml-4"),
				hx.Swap("outerHTML"),
				g.Text("- "+decodedValue),
			),
			g.Group(childNodes),
		)
	}

	// Start rendering from the first (outermost) node
	return renderNode(0)
}
