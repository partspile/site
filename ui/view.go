package ui

// View types
const (
	ViewList int = iota + 1
	ViewGrid
	ViewTree
)

func StringToView(s string) int {
	switch s {
	case "list":
		return ViewList
	case "grid":
		return ViewGrid
	case "tree":
		return ViewTree
	}
	return ViewList
}
