package ads

import (
	"bytes"
	"strings"
	"testing"

	"github.com/partspile/site/internal/facet"
	g "maragu.dev/gomponents"
)

func renderNode(t *testing.T, n g.Node) string {
	t.Helper()
	var buf bytes.Buffer
	if err := n.Render(&buf); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestAdPriceRowModes(t *testing.T) {
	d, ok := facet.Get("price")
	if !ok {
		t.Fatal("price facet missing")
	}
	cfg := NewFormConfig(facet.FormDefaults{Currency: "USD"})

	priced := renderNode(t, AdPriceRow(cfg, d, cfg.Defaults, PriceRowView{}))
	freeAt := strings.Index(priced, "List as FREE")
	wantedAt := strings.Index(priced, ">Wanted<")
	amountAt := strings.Index(priced, `type="number"`)
	if freeAt < 0 || wantedAt < freeAt || amountAt < wantedAt {
		t.Fatalf("priced row order free=%d wanted=%d amount=%d",
			freeAt, wantedAt, amountAt)
	}

	wanted := renderNode(t, AdPriceRow(cfg, d, cfg.Defaults, PriceRowView{
		IsWanted: true,
	}))
	if !strings.Contains(wanted, `value="-1"`) {
		t.Fatal("wanted row should store price -1")
	}
	if strings.Contains(wanted, `type="number"`) {
		t.Fatal("wanted row should hide the amount field")
	}
	if !strings.Contains(wanted, `name="price_wanted"`) {
		t.Fatal("wanted row missing checkbox")
	}

	free := renderNode(t, AdPriceRow(cfg, d, cfg.Defaults, PriceRowView{
		IsFree: true,
	}))
	if strings.Contains(free, `type="number"`) ||
		!strings.Contains(free, `value="0"`) {
		t.Fatal("free row should hide the amount and store 0")
	}
}
