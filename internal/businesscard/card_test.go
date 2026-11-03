package businesscard_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/partspile/site/internal/businesscard"
	"github.com/partspile/site/internal/config"
)

func TestSiteURL(t *testing.T) {
	got := businesscard.SiteURL("parts-pile.com")
	want := "https://parts-pile.com/"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSiteURLUsesHTTPS(t *testing.T) {
	url := businesscard.SiteURL("parts-pile.com")
	if !strings.HasPrefix(url, "https://") {
		t.Fatalf("expected https URL, got %q", url)
	}
}

func TestRenderPrintCard(t *testing.T) {
	prev := config.ServerName
	config.ServerName = "Parts Pile"
	t.Cleanup(func() { config.ServerName = prev })

	img, err := businesscard.RenderPrintCard(businesscard.PrintCardOptions{
		Host:       "parts-pile.com",
		ImagesRoot: businesscard.ResolveRepoPath("static/images"),
	})
	if err != nil {
		t.Fatalf("RenderPrintCard: %v", err)
	}

	b := img.Bounds()
	if b.Dx() != 1125 || b.Dy() != 675 {
		t.Fatalf("expected 1125x675 (3.75\"x2.25\" @ 300 DPI), got %dx%d",
			b.Dx(), b.Dy())
	}

	var buf bytes.Buffer
	if err := businesscard.WritePNG(&buf, img); err != nil {
		t.Fatalf("WritePNG: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte{0x89, 'P', 'N', 'G'}) {
		t.Fatal("expected PNG output")
	}

	decoded, err := png.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("png decode: %v", err)
	}
	if decoded.Bounds().Dx() != b.Dx() {
		t.Fatal("decoded PNG size mismatch")
	}

	bg := countColorNear(img, 0x18, 0x18, 0x1B)
	if bg < 1000 {
		t.Fatalf("expected dark card background, got %d matching pixels", bg)
	}
}

func countColorNear(img *image.RGBA, r, g, b uint8) int {
	bounds := img.Bounds()
	n := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if near(c, r, g, b, 8) {
				n++
			}
		}
	}
	return n
}

func near(c color.RGBA, r, g, b, tol uint8) bool {
	dr := absDiff(c.R, r)
	dg := absDiff(c.G, g)
	db := absDiff(c.B, b)
	return dr <= tol && dg <= tol && db <= tol
}

func absDiff(a, b uint8) uint8 {
	if a > b {
		return a - b
	}
	return b - a
}
