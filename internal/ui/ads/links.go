package ads

import (
	"github.com/partspile/site/internal/ad"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

const descriptionLinkClass = "text-blue-600 dark:text-blue-400 hover:underline"

// DescriptionTextWithLinks renders description text with https URLs linked.
func DescriptionTextWithLinks(text string) g.Node {
	segments := ad.SplitDescriptionLinks(text)
	if len(segments) == 1 && segments[0].Link == nil {
		return g.Text(text)
	}
	nodes := make([]g.Node, 0, len(segments))
	for _, seg := range segments {
		if seg.Link != nil {
			nodes = append(nodes, A(
				Href(seg.Link.URL),
				Class(descriptionLinkClass),
				g.Attr("rel", "noopener noreferrer"),
				g.Attr("target", "_blank"),
				g.Text(seg.Link.Text),
			))
			continue
		}
		if seg.Text != "" {
			nodes = append(nodes, g.Text(seg.Text))
		}
	}
	return g.Group(nodes)
}
