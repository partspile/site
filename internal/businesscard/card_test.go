package businesscard_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/partspile/site/internal/businesscard"
)

func TestLoadCategoriesAlphabeticalIDs(t *testing.T) {
	cats, err := businesscard.LoadCategories("")
	if err != nil {
		t.Fatalf("LoadCategories: %v", err)
	}
	if len(cats) != 9 {
		t.Fatalf("expected 9 categories, got %d", len(cats))
	}
	if cats[0].Name != "Agricultural Equipment" || cats[0].ID != 1 {
		t.Fatalf("unexpected first category: %+v", cats[0])
	}
	if cats[3].Name != "Bicycles" || cats[3].ID != 4 {
		t.Fatalf("unexpected bicycles category: %+v", cats[3])
	}
}

func TestCategoryURL(t *testing.T) {
	got := businesscard.CategoryURL("parts-pile.com", 4)
	want := "https://parts-pile.com/c/4"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRenderPrintCard(t *testing.T) {
	cats, err := businesscard.LoadCategories("")
	if err != nil {
		t.Fatalf("LoadCategories: %v", err)
	}
	cat, err := businesscard.FindCategory(cats, "Bicycles")
	if err != nil {
		t.Fatalf("FindCategory: %v", err)
	}

	img, err := businesscard.RenderPrintCard(businesscard.PrintCardOptions{
		Category:   cat,
		Host:       "parts-pile.com",
		ImagesRoot: businesscard.ResolveRepoPath("static/images/category"),
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
		t.Fatalf("expected dark zinc-900 background, got %d pixels", bg)
	}
	light := countColorNear(img, 0xE4, 0xE4, 0xE7)
	if light < 500 {
		t.Fatalf("expected zinc-200 content, got %d pixels", light)
	}
	blue := countColorNear(img, 0x60, 0xA5, 0xFA)
	if blue < 100 {
		t.Fatalf("expected category pill border, got %d blue pixels", blue)
	}
	pillBG := countColorNear(img, 0x1E, 0x3A, 0x8A)
	if pillBG < 100 {
		t.Fatalf("expected category pill fill, got %d blue-900 pixels", pillBG)
	}
}

func TestRenderPrintCardAllCategories(t *testing.T) {
	cats, err := businesscard.LoadCategories("")
	if err != nil {
		t.Fatalf("LoadCategories: %v", err)
	}
	for _, cat := range cats {
		name := cat.Name
		t.Run(name, func(t *testing.T) {
			_, err := businesscard.RenderPrintCard(businesscard.PrintCardOptions{
				Category:   cat,
				Host:       "parts-pile.com",
				ImagesRoot: businesscard.ResolveRepoPath("static/images/category"),
			})
			if err != nil {
				t.Fatalf("RenderPrintCard: %v", err)
			}
		})
	}
}

func TestLongCategoryNameFits(t *testing.T) {
	cats, err := businesscard.LoadCategories("")
	if err != nil {
		t.Fatalf("LoadCategories: %v", err)
	}
	cat, err := businesscard.FindCategory(cats, "Agricultural Equipment Parts")
	if err != nil {
		t.Fatalf("FindCategory: %v", err)
	}
	_, err = businesscard.RenderPrintCard(businesscard.PrintCardOptions{
		Category:   cat,
		Host:       "parts-pile.com",
		ImagesRoot: businesscard.ResolveRepoPath("static/images/category"),
	})
	if err != nil {
		t.Fatalf("RenderPrintCard: %v", err)
	}
}

func countColorNear(img *image.RGBA, r, g, b uint8) int {
	n := 0
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.RGBAAt(x, y)
			if colorNear(c, r, g, b) {
				n++
			}
		}
	}
	return n
}

func colorNear(c color.RGBA, r, g, b uint8) bool {
	const tol = 20
	return abs(int(c.R)-int(r)) <= tol &&
		abs(int(c.G)-int(g)) <= tol &&
		abs(int(c.B)-int(b)) <= tol &&
		c.A > 200
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestCategoryURLUsesHTTPS(t *testing.T) {
	url := businesscard.CategoryURL("parts-pile.com", 1)
	if !strings.HasPrefix(url, "https://") {
		t.Fatalf("expected https URL, got %q", url)
	}
}
