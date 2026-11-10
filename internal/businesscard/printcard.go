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
	// Site dark mode: body bg-zinc-900. Reading text and the QR use
	// --content-ink (#f4efe6), the warm parchment used instead of zinc gray.
	cardBG    = color.RGBA{R: 0x18, G: 0x18, B: 0x1B, A: 0xFF}
	brandText = color.RGBA{R: 0xF4, G: 0xEF, B: 0xE6, A: 0xFF}
	// About-page block.svg red filter approximation.
	noIconRed = color.RGBA{R: 0xDC, G: 0x26, B: 0x26, A: 0xFF}
)

// PrintCardOptions configures flat business card image generation.
type PrintCardOptions struct {
	Host       string
	ImagesRoot string // directory with the logo PNG and block.svg
}

// RenderPrintCard draws a Staples-ready business card at 300 DPI:
// 3.75"×2.25" canvas (with 0.125" bleed), content inside the 3.5"×2" trim
// and an additional 0.125" safe inset from the trim edge.
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
	contentPad := inchesToPx(safeInsetIn)

	canvas := image.NewRGBA(image.Rect(0, 0, totalW, totalH))
	draw.Draw(canvas, canvas.Bounds(), &image.Uniform{C: cardBG}, image.Point{}, draw.Src)

	trim := image.Rect(bleedPx, bleedPx, bleedPx+trimW, bleedPx+trimH)

	const cssScale = 3.0
	px := func(cssPx float64) int {
		return int(cssPx*cssScale + 0.5)
	}
	colGap := px(20)
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
	// Cap height so the pitch still fits under the logo.
	contentH := trimH - 2*contentPad
	maxLogoH := int(float64(contentH)*0.40 + 0.5)
	logoPath := filepath.Join(opts.ImagesRoot, "parts-pile-logo-dark.png")
	logo, err := loadPNGScaledToFit(logoPath, leftMaxW, maxLogoH)
	if err != nil {
		return nil, fmt.Errorf("render logo: %w", err)
	}
	logoH := logo.Bounds().Dy()
	leftTop := trim.Min.Y + contentPad
	drawAt(canvas, logo, contentLeft, leftTop)

	bodyTop := leftTop + logoH + bodyGap

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

	blockPath := filepath.Join(opts.ImagesRoot, "block.svg")
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
	qrURL := SiteURL(opts.Host)
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
