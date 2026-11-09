package ad

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/partspile/site/internal/entrylog"
	"github.com/partspile/site/internal/imagestore"
)

// ParseImageOrder parses a comma-separated 1-based permutation of 1..n.
// Empty input means no reorder (nil, nil).
func ParseImageOrder(raw string, n int) ([]int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if n < 1 {
		return nil, fmt.Errorf("no images to reorder")
	}
	parts := strings.Split(raw, ",")
	order := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid image order")
		}
		order = append(order, idx)
	}
	if err := imagestore.ValidateReorderOrder(order, n); err != nil {
		return nil, err
	}
	return order, nil
}

// RemapDescriptionImageIndices rewrites "Images Added" history bodies so
// each stored index follows the same permutation as ReorderAdImages.
func RemapDescriptionImageIndices(desc string, order []int) string {
	if len(order) == 0 || imagestore.IdentityOrder(order) {
		return desc
	}
	oldToNew := make(map[int]int, len(order))
	for newPos, oldIdx := range order {
		oldToNew[oldIdx] = newPos + 1
	}
	blocks := entrylog.Parse(desc)
	if len(blocks) == 0 {
		return desc
	}
	changed := false
	for i := range blocks {
		if blocks[i].Label != imagesAddedLabel {
			continue
		}
		body, ok := remapImageIndexBody(blocks[i].Body, oldToNew)
		if !ok {
			continue
		}
		blocks[i].Body = body
		changed = true
	}
	if !changed {
		return desc
	}
	return entrylog.Join(blocks, nil)
}

func remapImageIndexBody(body string, oldToNew map[int]int) (string, bool) {
	body = strings.TrimSpace(body)
	if body == "" {
		return body, false
	}
	parts := strings.Split(body, ",")
	out := make([]string, 0, len(parts))
	changed := false
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 {
			out = append(out, part)
			continue
		}
		if mapped, ok := oldToNew[n]; ok {
			if mapped != n {
				changed = true
			}
			out = append(out, strconv.Itoa(mapped))
			continue
		}
		out = append(out, part)
	}
	if !changed {
		return body, false
	}
	return strings.Join(out, ","), true
}
