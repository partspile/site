package businesscard

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"path/filepath"

	"github.com/partspile/site/internal/config"
	"github.com/skip2/go-qrcode"
)

const (
	printDPI = 300
	// Staples.com upload: full artwork 3.75"×2.25", finished cut 3.5"×2".
	trimWidthIn  = 3.5
	trimHeightIn = 2.0
	bleedIn      = 0.125 // extends file to 3.75"×2.25"
	safeInsetIn  = 0.125 // Staples safe zone inside the trim
	marginIn     = 0.125 // extra inset beyond the safe zone
)

var (
	// Site dark mode: body bg-zinc-900. Category lines, the QR, and the
	// hostname use the logo's gold. brandText is --content-ink (#f4efe6).
	cardBG    = color.RGBA{R: 0x18, G: 0x18, B: 0x1B, A: 0xFF}
	brandText = color.RGBA{R: 0xF4, G: 0xEF, B: 0xE6, A: 0xFF}
	logoGold  = color.RGBA{R: 0xEB, G: 0xD3, B: 0x9B, A: 0xFF}
)

// PrintCardOptions configures flat business card image generation.
type PrintCardOptions struct {
	Host       string
	ImagesRoot string // logo PNG and category icons
}

// RenderPrintCard draws a Staples-ready business card at 300 DPI:
// 3.75"×2.25" canvas (with 0.125" bleed), content inside the 3.5"×2" trim
// and 0.25" in from the trim edge (0.125" safe zone plus 0.125" margin).
func RenderPrintCard(opts PrintCardOptions) (*image.RGBA, error) {
	if opts.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if opts.ImagesRoot == "" {
		opts.ImagesRoot = "static/images"
	}

	totalW := inchesToPx(trimWidthIn + 2*bleedIn)
	totalH := inchesToPx(trimHeightIn + 2*bleedIn)
	trimW := inchesToPx(trimWidthIn)
	trimH := inchesToPx(trimHeightIn)
	bleedPx := inchesToPx(bleedIn)
	contentPad := inchesToPx(safeInsetIn + marginIn)

	canvas := image.NewRGBA(image.Rect(0, 0, totalW, totalH))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: cardBG}, image.Point{}, draw.Src)

	trim := image.Rect(bleedPx, bleedPx, bleedPx+trimW, bleedPx+trimH)

	const cssScale = 3.0
	px := func(cssPx float64) int {
		return int(cssPx*cssScale + 0.5)
	}
	colGap := px(36)
	bodyGap := px(18)

	contentLeft := trim.Min.X + contentPad
	contentRight := trim.Max.X - contentPad

	hostLabel := opts.Host
	hostSize := 11.0 * cssScale * 72 / printDPI
	hostGap := px(8)
	hostH, err := textHeight(hostLabel, hostSize, fontRegular)
	if err != nil {
		return nil, err
	}

	qrSize := int(float64(trimH)*0.48 + 0.5)
	maxQRH := trimH - 2*contentPad - hostGap - hostH
	if qrSize > maxQRH {
		qrSize = maxQRH
	}
	qrX := contentRight - qrSize
	leftMaxW := qrX - colGap - contentLeft
	if leftMaxW < 120 {
		qrSize = trimW/2 - contentPad
		if qrSize > maxQRH {
			qrSize = maxQRH
		}
		qrX = contentRight - qrSize
		leftMaxW = qrX - colGap - contentLeft
	}

	// Dark card matches site dark mode, so use the dark-background wordmark.
	// Cap height so the categories still fit under the logo.
	contentH := trimH - 2*contentPad
	maxLogoH := int(float64(contentH)*0.40 + 0.5)
	logoPath := filepath.Join(opts.ImagesRoot, "parts-pile-logo-dark.png")
	logo, err := loadPNGScaledToFit(logoPath, leftMaxW, maxLogoH)
	if err != nil {
		return nil, fmt.Errorf("render logo: %w", err)
	}
	logoW := logo.Bounds().Dx()
	logoH := logo.Bounds().Dy()

	type cardCategory struct {
		label string
		icon  string
	}
	categories := []cardCategory{
		{"Cars & Trucks and Parts", "car.svg"},
		{"Motorcycles and Parts", "motorcycle.svg"},
		{"Bicycles and Parts", "bicycle.svg"},
		{"Ag Equipment and Parts", "ag.svg"},
	}

	bodyBottom := trim.Max.Y - contentPad
	availBodyH := bodyBottom - (trim.Min.Y + contentPad + logoH + bodyGap)

	listSize := 11.0 * cssScale * 72 / printDPI
	iconGap := px(6)

	var (
		listLineH int
		icons     []*image.RGBA
		rowW      int
	)
	for try := 0; try < 12; try++ {
		var err error
		listLineH, err = textHeight("Ay", listSize, fontRegular)
		if err != nil {
			return nil, err
		}
		// Same viewBox square for every icon. Matching ink height
		// enlarges the motorcycle, which sits shorter in that grid.
		// The square includes padding, so it has to be taller than
		// the label for the artwork to read larger than the text.
		iconSz := listLineH * 7 / 4
		if iconSz < 1 {
			iconSz = 1
		}
		// Drop the extra gap between lines, but keep a full icon
		// square so the rows do not collide.
		rowH := iconSz
		if rowH < listLineH {
			rowH = listLineH
		}
		icons = make([]*image.RGBA, 0, len(categories))
		for _, cat := range categories {
			path := filepath.Join(
				opts.ImagesRoot, "category", cat.icon,
			)
			img, err := renderSVGRGBA(path, iconSz, logoGold)
			if err != nil {
				return nil, fmt.Errorf("render %s: %w", cat.icon, err)
			}
			icons = append(icons, img)
		}
		rowW = 0
		for _, cat := range categories {
			w, err := textWidth(cat.label, listSize, fontRegular)
			if err != nil {
				return nil, err
			}
			row := iconSz + iconGap + w
			if row > rowW {
				rowW = row
			}
		}
		listH := len(categories) * rowH
		if listH <= availBodyH && rowW <= leftMaxW {
			break
		}
		listSize *= 0.92
		iconGap = max(2, int(float64(iconGap)*0.92))
	}

	// Logo and category icons share one left edge. Center the logo
	// column and the QR together so the side margins match.
	groupW := logoW
	if rowW > groupW {
		groupW = rowW
	}
	hostW, err := textWidth(hostLabel, hostSize, fontRegular)
	if err != nil {
		return nil, err
	}
	rightW := qrSize
	if hostW > rightW {
		rightW = hostW
	}
	blockW := groupW + colGap + rightW
	span := contentRight - contentLeft
	originX := contentLeft
	if blockW < span {
		originX += (span - blockW) / 2
	}
	groupX := originX
	qrX = originX + groupW + colGap
	if rightW > qrSize {
		qrX += (rightW - qrSize) / 2
	}

	leftTop := trim.Min.Y + contentPad
	drawAt(canvas, logo, groupX, leftTop)

	y := leftTop + logoH + bodyGap
	iconSz := icons[0].Bounds().Dx()
	rowH := iconSz
	if rowH < listLineH {
		rowH = listLineH
	}
	textX := groupX + iconSz + iconGap
	for i, cat := range categories {
		icon := icons[i]
		iconY := y + (rowH-icon.Bounds().Dy())/2
		drawAt(canvas, icon, groupX, iconY)
		itemBaseline, err := textBaselineForCenterY(
			cat.label, y+rowH/2, listSize, fontRegular,
		)
		if err != nil {
			return nil, err
		}
		if err := drawLeftText(
			canvas, cat.label, textX, itemBaseline,
			listSize, fontRegular, logoGold,
		); err != nil {
			return nil, err
		}
		y += rowH
	}

	qrBlockH := qrSize + hostGap + hostH
	qrY := trim.Min.Y + (trimH-qrBlockH)/2
	if qrY < trim.Min.Y+contentPad {
		qrY = trim.Min.Y + contentPad
	}
	qrURL := SiteURL(opts.Host)
	qr, err := renderQRRGBA(qrURL, qrSize, logoGold)
	if err != nil {
		return nil, err
	}
	drawAt(canvas, qr, qrX, qrY)

	hostBaseline, err := textBaselineFromTop(
		hostLabel, qrY+qrSize+hostGap, hostSize, fontRegular,
	)
	if err != nil {
		return nil, err
	}
	if err := drawCenteredText(
		canvas, hostLabel, qrX+qrSize/2, hostBaseline,
		hostSize, fontRegular, logoGold,
	); err != nil {
		return nil, err
	}

	return canvas, nil
}

// RenderPrintCardBack draws the card back at the same size and background
// as the front. The tagline is the bold gold headline. The about-page
// points sit under it, smaller and centered as one column.
func RenderPrintCardBack() (*image.RGBA, error) {
	totalW := inchesToPx(trimWidthIn + 2*bleedIn)
	totalH := inchesToPx(trimHeightIn + 2*bleedIn)
	trimW := inchesToPx(trimWidthIn)
	trimH := inchesToPx(trimHeightIn)
	bleedPx := inchesToPx(bleedIn)
	contentPad := inchesToPx(safeInsetIn + marginIn)

	canvas := image.NewRGBA(image.Rect(0, 0, totalW, totalH))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: cardBG}, image.Point{}, draw.Src)

	const cssScale = 3.0
	px := func(cssPx float64) int {
		return int(cssPx*cssScale + 0.5)
	}
	tagline := "Buy - Sell - Trade"
	tagSize := 18.0 * cssScale * 72 / printDPI
	bodySize := 9.5 * cssScale * 72 / printDPI
	gap := px(20)

	contentLeft := bleedPx + contentPad
	contentRight := bleedPx + trimW - contentPad
	contentTop := bleedPx + contentPad
	contentBottom := bleedPx + trimH - contentPad

	columnMax := inchesToPx(2.3)
	if span := contentRight - contentLeft; span < columnMax {
		columnMax = span
	}
	block, err := layoutCardBackBullets(columnMax, bodySize)
	if err != nil {
		return nil, err
	}
	legalSize := 7.5 * cssScale * 72 / printDPI
	legalH, err := textHeight(config.LegalName, legalSize, fontRegular)
	if err != nil {
		return nil, err
	}
	legalGap := px(12)
	legalTop := contentBottom - legalH
	stackBottom := legalTop - legalGap

	tagH, err := textHeight(tagline, tagSize, fontBold)
	if err != nil {
		return nil, err
	}
	stackH := tagH + gap + block.height
	top := contentTop + (stackBottom-contentTop-stackH)/2
	if top < contentTop {
		top = contentTop
	}

	baseline, err := textBaselineFromTop(tagline, top, tagSize, fontBold)
	if err != nil {
		return nil, err
	}
	if err := drawCenteredText(
		canvas, tagline, totalW/2, baseline,
		tagSize, fontBold, logoGold,
	); err != nil {
		return nil, err
	}

	tagBottom, err := textBottomY(tagline, baseline, tagSize, fontBold)
	if err != nil {
		return nil, err
	}
	colLeft := totalW/2 - block.width/2
	if err := drawCardBackBullets(
		canvas, block, colLeft, tagBottom+gap, stackBottom,
	); err != nil {
		return nil, err
	}
	legalBase, err := textBaselineFromTop(
		config.LegalName, legalTop, legalSize, fontRegular,
	)
	if err != nil {
		return nil, err
	}
	if err := drawCenteredText(
		canvas, config.LegalName, totalW/2, legalBase,
		legalSize, fontRegular, brandText,
	); err != nil {
		return nil, err
	}
	return canvas, nil
}

func cardBackBullets() []string {
	return []string{
		"Posting is free.",
		"We don't sell your information or share it with " +
			"third parties for their own use.",
		"A phone number is all we ask for. " +
			"We keep your number private.",
	}
}

type cardBackBlock struct {
	items  [][]string
	size   float64
	textX  int
	width  int
	height int
}

func layoutCardBackBullets(maxW int, size float64) (cardBackBlock, error) {
	const cssScale = 3.0
	px := func(cssPx float64) int {
		return int(cssPx*cssScale + 0.5)
	}
	var block cardBackBlock
	block.size = size
	markW, err := textWidth("•", size, fontRegular)
	if err != nil {
		return block, err
	}
	gap := px(6)
	block.textX = markW + gap
	textMax := maxW - block.textX
	if textMax < 80 {
		return block, fmt.Errorf("card back is too narrow for the points")
	}

	lineGap := px(2)
	itemGap := px(7)
	widest := 0
	for n, point := range cardBackBullets() {
		lines, err := wrapLines(point, textMax, size, fontRegular)
		if err != nil {
			return block, err
		}
		block.items = append(block.items, lines)
		for _, line := range lines {
			w, err := textWidth(line, size, fontRegular)
			if err != nil {
				return block, err
			}
			if w > widest {
				widest = w
			}
			h, err := textHeight(line, size, fontRegular)
			if err != nil {
				return block, err
			}
			block.height += h + lineGap
		}
		if n < len(cardBackBullets())-1 {
			block.height += itemGap
		}
	}
	if block.height > lineGap {
		block.height -= lineGap
	}
	block.width = block.textX + widest
	return block, nil
}

func drawCardBackBullets(
	canvas *image.RGBA,
	block cardBackBlock,
	left, top, bottom int,
) error {
	const cssScale = 3.0
	px := func(cssPx float64) int {
		return int(cssPx*cssScale + 0.5)
	}
	lineGap := px(2)
	itemGap := px(7)
	y := top
	for n, lines := range block.items {
		for i, line := range lines {
			base, err := textBaselineFromTop(
				line, y, block.size, fontRegular,
			)
			if err != nil {
				return err
			}
			if i == 0 {
				if err := drawLeftText(
					canvas, "•", left, base,
					block.size, fontRegular, brandText,
				); err != nil {
					return err
				}
			}
			if err := drawLeftText(
				canvas, line, left+block.textX, base,
				block.size, fontRegular, brandText,
			); err != nil {
				return err
			}
			inkBottom, err := textBottomY(
				line, base, block.size, fontRegular,
			)
			if err != nil {
				return err
			}
			if inkBottom > bottom {
				return fmt.Errorf("card back points overflow")
			}
			y = inkBottom + lineGap
		}
		if n < len(block.items)-1 {
			y += itemGap
		}
	}
	return nil
}

// WritePNG encodes img as a PNG with alpha transparency.
func WritePNG(w io.Writer, img image.Image) error {
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	return enc.Encode(w, img)
}

func inchesToPx(inches float64) int {
	return int(inches*printDPI + 0.5)
}

func drawAt(dst *image.RGBA, src image.Image, x, y int) {
	draw.Draw(dst, image.Rect(x, y, x+src.Bounds().Dx(), y+src.Bounds().Dy()),
		src, src.Bounds().Min, draw.Over)
}

func renderQRRGBA(url string, size int, fg color.Color) (*image.RGBA, error) {
	code, err := qrcode.New(url, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	code.DisableBorder = true
	img := code.Image(size)
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := color.GrayModel.Convert(img.At(x, y)).(color.Gray)
			if c.Y < 128 {
				rgba.Set(x, y, fg)
			}
		}
	}
	return rgba, nil
}
