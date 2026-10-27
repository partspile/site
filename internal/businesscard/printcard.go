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
	safeInsetIn  = 0.125 // keep content inside trim (Staples safe zone)
)

var (
	// Site dark mode: body bg-zinc-900, text-zinc-200.
	cardBG    = color.RGBA{R: 0x18, G: 0x18, B: 0x1B, A: 0xFF}
	brandText = color.RGBA{R: 0xE4, G: 0xE4, B: 0xE7, A: 0xFF}
	// Category pill: dark:border-blue-400 dark:bg-blue-900.
	pillBorder = color.RGBA{R: 0x60, G: 0xA5, B: 0xFA, A: 0xFF}
	pillFill   = color.RGBA{R: 0x1E, G: 0x3A, B: 0x8A, A: 0xFF}
	// Category icons: dark:invert dark:opacity-80 (NRGBA; Go RGBA is premul).
	pillIcon = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xCC}
	// About-page block.svg red filter approximation.
	noIconRed = color.RGBA{R: 0xDC, G: 0x26, B: 0x26, A: 0xFF}
)

// PrintCardOptions configures flat business card image generation.
type PrintCardOptions struct {
	Category   Category
	Host       string
	ImagesRoot string
}

// RenderPrintCard draws a Staples-ready business card at 300 DPI:
// 3.75"×2.25" canvas (with 0.125" bleed), content inside the 3.5"×2" trim
// and an additional 0.125" safe inset from the trim edge.
func RenderPrintCard(opts PrintCardOptions) (*image.RGBA, error) {
	if opts.Host == "" {
		return nil, fmt.Errorf("host is required")
	}
	if opts.ImagesRoot == "" {
		opts.ImagesRoot = "static/images/category"
	}

	totalW := inchesToPx(trimWidthIn + 2*bleedIn)
	totalH := inchesToPx(trimHeightIn + 2*bleedIn)
	trimW := inchesToPx(trimWidthIn)
	trimH := inchesToPx(trimHeightIn)
	bleedPx := inchesToPx(bleedIn)
	contentPad := inchesToPx(safeInsetIn)

	canvas := image.NewRGBA(image.Rect(0, 0, totalW, totalH))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: cardBG}, image.Point{}, draw.Src)

	trim := image.Rect(bleedPx, bleedPx, bleedPx+trimW, bleedPx+trimH)

	// Match site homepage visual proportions: icon ink slightly taller than
	// adjacent capitals (nav rock vs site title; pill icon vs category).
	const cssScale = 3.0
	px := func(cssPx float64) int {
		return int(cssPx*cssScale + 0.5)
	}
	titleSize := 20.0 * cssScale * 72 / printDPI
	categorySize := 16.0 * cssScale * 72 / printDPI
	logoGap := px(8)
	pillIconGap := px(8)
	pillPadX := px(20)
	pillPadY := px(8)
	pillBorderW := px(2)
	colGap := px(20)
	stackGap := px(10)
	bodyGap := px(18)

	titleText := config.ServerName
	const (
		iconToText    = 1.35 // CSS boxes are larger than ink; bias icons up
		titleHeavy    = 1    // light embolden for title
		categoryHeavy = 0    // bold only for category
	)

	contentLeft := trim.Min.X + contentPad
	contentRight := trim.Max.X - contentPad

	hostLabel := opts.Host
	hostSize := 11.0 * cssScale * 72 / printDPI
	hostGap := px(8)
	hostH, err := textHeight(hostLabel, hostSize, fontRegular)
	if err != nil {
		return nil, err
	}

	// QR on the right; leave room for the left column and host label.
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

	titleH, err := textHeight(titleText, titleSize, fontBold)
	if err != nil {
		return nil, err
	}
	catH, err := textHeight(opts.Category.Name, categorySize, fontBold)
	if err != nil {
		return nil, err
	}
	rockTargetH := int(float64(titleH)*iconToText + 0.5)
	iconTargetH := int(float64(catH)*iconToText + 0.5)

	rockPath := filepath.Join(filepath.Dir(opts.ImagesRoot), "rock.svg")
	rock, err := renderSVGScaledToInkHeight(rockPath, rockTargetH, nil)
	if err != nil {
		return nil, fmt.Errorf("render rock icon: %w", err)
	}
	rockSize := rock.Bounds().Dy()
	rockW := rock.Bounds().Dx()

	titleW, err := textWidth(titleText, titleSize, fontBold)
	if err != nil {
		return nil, err
	}
	logoW := rockW + logoGap + titleW + 2*titleHeavy

	// Shrink category text until the pill fits the left column.
	var (
		icon         *image.RGBA
		pillIconSize int
		catW         int
		pillH        int
		pillW        int
	)
	for try := 0; try < 8; try++ {
		catH, err = textHeight(opts.Category.Name, categorySize, fontBold)
		if err != nil {
			return nil, err
		}
		iconTargetH = int(float64(catH)*iconToText + 0.5)
		iconPath := filepath.Join(opts.ImagesRoot, opts.Category.ImageFile)
		icon, err = renderSVGScaledToInkHeight(iconPath, iconTargetH, pillIcon)
		if err != nil {
			return nil, fmt.Errorf("render icon %s: %w", iconPath, err)
		}
		pillIconSize = icon.Bounds().Dy()
		catW, err = textWidth(opts.Category.Name, categorySize, fontBold)
		if err != nil {
			return nil, err
		}
		pillInnerH := pillIconSize
		if catH > pillInnerH {
			pillInnerH = catH
		}
		pillH = pillInnerH + 2*pillPadY
		pillW = icon.Bounds().Dx() + pillIconGap + catW + 2*pillPadX +
			2*categoryHeavy
		if pillW <= leftMaxW && logoW <= leftMaxW {
			break
		}
		categorySize *= 0.9
		if logoW > leftMaxW {
			titleSize *= 0.9
			titleH, err = textHeight(titleText, titleSize, fontBold)
			if err != nil {
				return nil, err
			}
			rockTargetH = int(float64(titleH)*iconToText + 0.5)
			rock, err = renderSVGScaledToInkHeight(rockPath, rockTargetH, nil)
			if err != nil {
				return nil, err
			}
			rockSize = rock.Bounds().Dy()
			rockW = rock.Bounds().Dx()
			titleW, err = textWidth(titleText, titleSize, fontBold)
			if err != nil {
				return nil, err
			}
			logoW = rockW + logoGap + titleW + 2*titleHeavy
		}
	}

	logoH := rockSize
	if titleH > logoH {
		logoH = titleH
	}
	leftTop := trim.Min.Y + contentPad

	logoCenterY := leftTop + logoH/2
	titleBaseline, err := textBaselineForCenterY(
		titleText, logoCenterY, titleSize, fontBold,
	)
	if err != nil {
		return nil, err
	}

	drawAt(canvas, rock, contentLeft, logoCenterY-rockSize/2)
	if err := drawLeftTextEmboldened(
		canvas, titleText, contentLeft+rockW+logoGap, titleBaseline,
		titleSize, fontBold, brandText, titleHeavy,
	); err != nil {
		return nil, err
	}

	pillTop := leftTop + logoH + stackGap
	pillRect := image.Rect(
		contentLeft, pillTop, contentLeft+pillW, pillTop+pillH,
	)
	fillRoundedRect(canvas, pillRect, pillH/2, pillBorder)
	inner := image.Rect(
		pillRect.Min.X+pillBorderW,
		pillRect.Min.Y+pillBorderW,
		pillRect.Max.X-pillBorderW,
		pillRect.Max.Y-pillBorderW,
	)
	fillRoundedRect(canvas, inner, (pillH-2*pillBorderW)/2, pillFill)

	pillCenterY := pillTop + pillH/2
	pillContentLeft := pillRect.Min.X + pillPadX
	iconW := icon.Bounds().Dx()
	drawAt(canvas, icon, pillContentLeft, pillCenterY-pillIconSize/2)

	catBaseline, err := textBaselineForCenterY(
		opts.Category.Name, pillCenterY, categorySize, fontBold,
	)
	if err != nil {
		return nil, err
	}
	if err := drawLeftTextEmboldened(
		canvas, opts.Category.Name,
		pillContentLeft+iconW+pillIconGap, catBaseline,
		categorySize, fontBold, brandText, categoryHeavy,
	); err != nil {
		return nil, err
	}

	bodyTop := pillTop + pillH + bodyGap

	aboutBlurb := "Remember classified ads? Post ad with your number and " +
		"folks contact you. " + config.ServerName +
		" works the same way—except your " +
		"number stays hidden."
	noItems := []string{
		"No email",
		"No Facebook friends",
		"No posting fees",
		"No credit cards",
	}

	bodyBottom := trim.Max.Y - contentPad
	availBodyH := bodyBottom - bodyTop

	bodySize := 10.0 * cssScale * 72 / printDPI
	listSize := 10.0 * cssScale * 72 / printDPI
	lineGap := px(2)
	paraGap := px(12)
	itemGap := px(3)
	noIconGap := px(4)

	blockPath := filepath.Join(filepath.Dir(opts.ImagesRoot), "block.svg")
	var (
		bodyLines []string
		lineH     int
		listLineH int
		noIconSz  int
		noIconImg *image.RGBA
	)
	for try := 0; try < 10; try++ {
		var err error
		bodyLines, err = wrapLines(aboutBlurb, leftMaxW, bodySize, fontRegular)
		if err != nil {
			return nil, err
		}
		lineH, err = textHeight("Ay", bodySize, fontRegular)
		if err != nil {
			return nil, err
		}
		listLineH, err = textHeight("Ay", listSize, fontBold)
		if err != nil {
			return nil, err
		}
		noIconSz = listLineH
		bodyH := len(bodyLines)*lineH + (len(bodyLines)-1)*lineGap
		if len(bodyLines) > 0 {
			bodyH += paraGap
		}
		listH := len(noItems)*noIconSz + (len(noItems)-1)*itemGap
		if bodyH+listH <= availBodyH {
			noIconImg, err = renderSVGScaledToInkHeight(
				blockPath, noIconSz, noIconRed,
			)
			if err != nil {
				return nil, fmt.Errorf("render block icon: %w", err)
			}
			break
		}
		bodySize *= 0.92
		listSize *= 0.92
		lineGap = max(1, int(float64(lineGap)*0.92))
		paraGap = max(2, int(float64(paraGap)*0.92))
		itemGap = max(1, int(float64(itemGap)*0.92))
		if try == 9 {
			noIconImg, err = renderSVGScaledToInkHeight(
				blockPath, max(8, noIconSz), noIconRed,
			)
			if err != nil {
				return nil, fmt.Errorf("render block icon: %w", err)
			}
		}
	}

	y := bodyTop
	for i, line := range bodyLines {
		baseline, err := textBaselineFromTop(
			line, y, bodySize, fontRegular,
		)
		if err != nil {
			return nil, err
		}
		if err := drawLeftText(
			canvas, line, contentLeft, baseline,
			bodySize, fontRegular, brandText,
		); err != nil {
			return nil, err
		}
		y += lineH
		if i < len(bodyLines)-1 {
			y += lineGap
		}
	}
	y += paraGap

	for _, item := range noItems {
		iconY := y + (noIconSz-noIconImg.Bounds().Dy())/2
		drawAt(canvas, noIconImg, contentLeft, iconY)
		itemBaseline, err := textBaselineForCenterY(
			item, y+noIconSz/2, listSize, fontBold,
		)
		if err != nil {
			return nil, err
		}
		if err := drawLeftText(
			canvas, item,
			contentLeft+noIconImg.Bounds().Dx()+noIconGap, itemBaseline,
			listSize, fontBold, brandText,
		); err != nil {
			return nil, err
		}
		y += noIconSz + itemGap
	}

	qrBlockH := qrSize + hostGap + hostH
	qrY := trim.Min.Y + (trimH-qrBlockH)/2
	if qrY < trim.Min.Y+contentPad {
		qrY = trim.Min.Y + contentPad
	}
	qrURL := CategoryURL(opts.Host, opts.Category.ID)
	qr, err := renderQRRGBA(qrURL, qrSize, brandText)
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
		hostSize, fontRegular, brandText,
	); err != nil {
		return nil, err
	}

	return canvas, nil
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
