package businesscard

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"strings"

	"github.com/partspile/site/internal/config"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type fontWeight int

const (
	fontBold fontWeight = iota
	fontRegular
)

func opentypeFace(size, dpi float64, weight fontWeight) (font.Face, error) {
	src := gobold.TTF
	if weight == fontRegular {
		src = goregular.TTF
	}
	f, err := opentype.Parse(src)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
}

func drawCenteredText(
	dst *image.RGBA,
	text string,
	centerX, baselineY int,
	size float64,
	weight fontWeight,
	clr color.Color,
) error {
	return drawCenteredTextEmboldened(
		dst, text, centerX, baselineY, size, weight, clr, 0,
	)
}

func drawCenteredTextEmboldened(
	dst *image.RGBA,
	text string,
	centerX, baselineY int,
	size float64,
	weight fontWeight,
	clr color.Color,
	embolden int,
) error {
	face, err := opentypeFace(size, printDPI, weight)
	if err != nil {
		return err
	}
	defer face.Close()

	bounds, _ := font.BoundString(face, text)
	textWidth := (bounds.Max.X - bounds.Min.X).Ceil()
	x := centerX - textWidth/2

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(clr),
		Face: face,
	}
	drawStringEmboldened(d, text, x, baselineY, embolden)
	return nil
}

func drawLeftText(
	dst *image.RGBA,
	text string,
	x, baselineY int,
	size float64,
	weight fontWeight,
	clr color.Color,
) error {
	return drawLeftTextEmboldened(
		dst, text, x, baselineY, size, weight, clr, 0,
	)
}

func drawLeftTextEmboldened(
	dst *image.RGBA,
	text string,
	x, baselineY int,
	size float64,
	weight fontWeight,
	clr color.Color,
	embolden int,
) error {
	face, err := opentypeFace(size, printDPI, weight)
	if err != nil {
		return err
	}
	defer face.Close()

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(clr),
		Face: face,
	}
	drawStringEmboldened(d, text, x, baselineY, embolden)
	return nil
}

func drawStringEmboldened(
	d *font.Drawer, text string, x, baselineY, radius int,
) {
	if radius < 0 {
		radius = 0
	}
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			d.Dot = fixed.Point26_6{
				X: fixed.I(x + dx),
				Y: fixed.I(baselineY + dy),
			}
			d.DrawString(text)
		}
	}
}

func textWidth(text string, size float64, weight fontWeight) (int, error) {
	face, err := opentypeFace(size, printDPI, weight)
	if err != nil {
		return 0, err
	}
	defer face.Close()

	bounds, _ := font.BoundString(face, text)
	return (bounds.Max.X - bounds.Min.X).Ceil(), nil
}

func wrapLines(
	text string, maxW int, size float64, weight fontWeight,
) ([]string, error) {
	// Keep multi-word brand phrases on one line when possible.
	protected := []struct{ from, to string }{
		{"phone number", "phone\u00a0number"},
	}
	if name := config.ServerName; name != "" {
		protected = append(protected, struct{ from, to string }{
			name, strings.ReplaceAll(name, " ", "\u00a0"),
		})
	}
	for _, p := range protected {
		text = strings.ReplaceAll(text, p.from, p.to)
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil, nil
	}
	for i := range words {
		words[i] = strings.ReplaceAll(words[i], "\u00a0", " ")
	}

	greedy, err := wrapGreedy(words, maxW, size, weight)
	if err != nil {
		return nil, err
	}
	best := greedy
	bestScore, err := raggedScore(best, size, weight)
	if err != nil {
		return nil, err
	}
	targetLines := len(greedy)

	// Try slightly narrower wraps to even out line lengths.
	for shrink := 2; shrink <= 24; shrink += 2 {
		limit := maxW * (100 - shrink) / 100
		if limit < maxW*6/10 {
			break
		}
		lines, err := wrapGreedy(words, limit, size, weight)
		if err != nil {
			return nil, err
		}
		if len(lines) > targetLines+1 {
			continue
		}
		score, err := raggedScore(lines, size, weight)
		if err != nil {
			return nil, err
		}
		// Prefer fewer lines when scores are close; otherwise less ragged.
		better := score < bestScore-maxW/40 ||
			(score <= bestScore && len(lines) <= len(best))
		if better {
			best = lines
			bestScore = score
		}
	}

	// Prefer sentence-ending breaks when a line is very short after a mid-phrase cut.
	best, err = preferSentenceBreaks(best, maxW, size, weight)
	if err != nil {
		return nil, err
	}
	return best, nil
}

func wrapGreedy(
	words []string, maxW int, size float64, weight fontWeight,
) ([]string, error) {
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		trial := cur + " " + w
		tw, err := textWidth(trial, size, weight)
		if err != nil {
			return nil, err
		}
		if tw <= maxW {
			cur = trial
			continue
		}
		lines = append(lines, cur)
		cur = w
	}
	lines = append(lines, cur)
	return lines, nil
}

func raggedScore(
	lines []string, size float64, weight fontWeight,
) (int, error) {
	if len(lines) == 0 {
		return 0, nil
	}
	minW, maxLineW := 0, 0
	for i, line := range lines {
		w, err := textWidth(line, size, weight)
		if err != nil {
			return 0, err
		}
		if i == 0 || w < minW {
			minW = w
		}
		if w > maxLineW {
			maxLineW = w
		}
	}
	return maxLineW - minW, nil
}

func preferSentenceBreaks(
	lines []string, maxW int, size float64, weight fontWeight,
) ([]string, error) {
	if len(lines) < 2 {
		return lines, nil
	}
	out := make([]string, 0, len(lines)+2)
	i := 0
	for i < len(lines) {
		line := lines[i]
		if i+1 < len(lines) {
			if idx := strings.LastIndex(line, ". "); idx > 0 {
				after := strings.TrimSpace(line[idx+2:])
				before := strings.TrimSpace(line[:idx+1])
				if after != "" && before != "" {
					next := after + " " + lines[i+1]
					tw, err := textWidth(next, size, weight)
					if err != nil {
						return nil, err
					}
					if tw <= maxW {
						out = append(out, before)
						lines[i+1] = next
						i++
						continue
					}
				}
			}
		}
		out = append(out, line)
		i++
	}
	return out, nil
}

func textBaselineForCenterY(
	text string, centerY int, size float64, weight fontWeight,
) (int, error) {
	face, err := opentypeFace(size, printDPI, weight)
	if err != nil {
		return 0, err
	}
	defer face.Close()

	bounds, _ := font.BoundString(face, text)
	midOffset := (bounds.Min.Y + bounds.Max.Y).Ceil() / 2
	return centerY - midOffset, nil
}

func textBottomY(
	text string, baselineY int, size float64, weight fontWeight,
) (int, error) {
	face, err := opentypeFace(size, printDPI, weight)
	if err != nil {
		return 0, err
	}
	defer face.Close()

	bounds, _ := font.BoundString(face, text)
	return baselineY + bounds.Max.Y.Ceil(), nil
}

func textHeight(text string, size float64, weight fontWeight) (int, error) {
	face, err := opentypeFace(size, printDPI, weight)
	if err != nil {
		return 0, err
	}
	defer face.Close()

	bounds, _ := font.BoundString(face, text)
	return bounds.Max.Y.Ceil() - bounds.Min.Y.Ceil(), nil
}

func textBaselineFromTop(
	text string, topY int, size float64, weight fontWeight,
) (int, error) {
	face, err := opentypeFace(size, printDPI, weight)
	if err != nil {
		return 0, err
	}
	defer face.Close()

	bounds, _ := font.BoundString(face, text)
	return topY - bounds.Min.Y.Ceil(), nil
}

func renderSVGRGBA(path string, sizePx int, tint color.Color) (*image.RGBA, error) {
	rgba, err := renderSVGNative(path, sizePx)
	if err != nil {
		return nil, err
	}
	return applyTint(rgba, tint), nil
}

// renderSVGScaledToInkHeight draws an SVG, crops transparent padding, and
// scales so the opaque ink height matches targetInkH (matching site visuals
// where icon ink ≈ adjacent capital-letter height).
func renderSVGScaledToInkHeight(
	path string, targetInkH int, tint color.Color,
) (*image.RGBA, error) {
	if targetInkH < 1 {
		targetInkH = 1
	}
	base := targetInkH * 4
	if base < 128 {
		base = 128
	}
	src, err := renderSVGNative(path, base)
	if err != nil {
		return nil, err
	}
	if tint != nil {
		src = applyTint(src, tint)
	}
	ink := opaqueBounds(src)
	if ink.Empty() {
		return nil, fmt.Errorf("no ink in %s", path)
	}
	cropped := image.NewRGBA(image.Rect(0, 0, ink.Dx(), ink.Dy()))
	draw.Draw(cropped, cropped.Bounds(), src, ink.Min, draw.Src)

	outW := int(float64(ink.Dx())*float64(targetInkH)/float64(ink.Dy()) + 0.5)
	if outW < 1 {
		outW = 1
	}
	out := image.NewRGBA(image.Rect(0, 0, outW, targetInkH))
	xdraw.CatmullRom.Scale(
		out, out.Bounds(), cropped, cropped.Bounds(), xdraw.Over, nil,
	)
	return out, nil
}

func applyTint(src *image.RGBA, tint color.Color) *image.RGBA {
	b := src.Bounds()
	out := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := src.At(x, y).RGBA()
			if a > 0x8000 {
				out.Set(x, y, tint)
			}
		}
	}
	return out
}

func opaqueBounds(src *image.RGBA) image.Rectangle {
	b := src.Bounds()
	minX, minY := b.Max.X, b.Max.Y
	maxX, maxY := b.Min.X, b.Min.Y
	found := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if src.RGBAAt(x, y).A > 128 {
				found = true
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x >= maxX {
					maxX = x + 1
				}
				if y >= maxY {
					maxY = y + 1
				}
			}
		}
	}
	if !found {
		return image.Rectangle{}
	}
	return image.Rect(minX, minY, maxX, maxY)
}

func renderSVGNative(path string, sizePx int) (*image.RGBA, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	icon, err := oksvg.ReadIconStream(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	// Stretch to the square like a site <img class="w-N h-N"> (object-fit: fill).
	vbW, vbH := icon.ViewBox.W, icon.ViewBox.H
	if vbW == 0 {
		vbW = float64(sizePx)
	}
	if vbH == 0 {
		vbH = float64(sizePx)
	}
	scaleX := float64(sizePx) / vbW
	scaleY := float64(sizePx) / vbH

	icon.Transform = rasterx.Identity.
		Scale(scaleX, scaleY).
		Translate(-icon.ViewBox.X, -icon.ViewBox.Y)

	rgba := image.NewRGBA(image.Rect(0, 0, sizePx, sizePx))
	draw.Draw(rgba, rgba.Bounds(), image.Transparent, image.Point{}, draw.Src)
	scanner := rasterx.NewScannerGV(sizePx, sizePx, rgba, rgba.Bounds())
	dasher := rasterx.NewDasher(sizePx, sizePx, scanner)
	icon.Draw(dasher, 1.0)
	return rgba, nil
}

func fillRoundedRect(
	dst *image.RGBA, r image.Rectangle, radius int, clr color.Color,
) {
	if r.Empty() {
		return
	}
	if radius < 0 {
		radius = 0
	}
	maxR := r.Dx() / 2
	if r.Dy()/2 < maxR {
		maxR = r.Dy() / 2
	}
	if radius > maxR {
		radius = maxR
	}

	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if inRoundedRect(x, y, r, radius) {
				dst.Set(x, y, clr)
			}
		}
	}
}

func inRoundedRect(x, y int, r image.Rectangle, radius int) bool {
	if x < r.Min.X || x >= r.Max.X || y < r.Min.Y || y >= r.Max.Y {
		return false
	}
	if radius == 0 {
		return true
	}

	cx, cy := x, y
	switch {
	case x < r.Min.X+radius && y < r.Min.Y+radius:
		cx, cy = r.Min.X+radius, r.Min.Y+radius
	case x >= r.Max.X-radius && y < r.Min.Y+radius:
		cx, cy = r.Max.X-1-radius, r.Min.Y+radius
	case x < r.Min.X+radius && y >= r.Max.Y-radius:
		cx, cy = r.Min.X+radius, r.Max.Y-1-radius
	case x >= r.Max.X-radius && y >= r.Max.Y-radius:
		cx, cy = r.Max.X-1-radius, r.Max.Y-1-radius
	default:
		return true
	}

	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= radius*radius
}
