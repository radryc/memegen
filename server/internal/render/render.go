// Package render draws meme captions onto template images on the server. It
// backs the on-demand image endpoint so memes can be stored as text specs.
package render

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"
	"sync"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	_ "image/gif"
	_ "image/jpeg"

	_ "golang.org/x/image/webp"
)

//go:embed Anton-Regular.ttf
var antonTTF []byte

const maxRenderSide = 1600

// Style mirrors the JSON stored in memes.style.
type Style struct {
	FontScale float64 `json:"fontScale"`
	Color     string  `json:"color"`
	Stroke    string  `json:"stroke"`
	Uppercase bool    `json:"uppercase"`
	Outline   bool    `json:"outline"`
}

// ParseStyle applies the same defaults as the client renderer.
func ParseStyle(raw string) Style {
	style := Style{
		FontScale: 1,
		Color:     "#ffffff",
		Stroke:    "#000000",
		Uppercase: true,
		Outline:   true,
	}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &style); err != nil {
			return Style{FontScale: 1, Color: "#ffffff", Stroke: "#000000", Uppercase: true, Outline: true}
		}
	}
	if style.FontScale < 0.4 || style.FontScale > 2.5 || math.IsNaN(style.FontScale) {
		style.FontScale = 1
	}
	if !validHex(style.Color) {
		style.Color = "#ffffff"
	}
	if !validHex(style.Stroke) {
		style.Stroke = "#000000"
	}
	return style
}

var (
	fontOnce sync.Once
	anton    *opentype.Font
	fontErr  error
)

func loadFont() (*opentype.Font, error) {
	fontOnce.Do(func() {
		anton, fontErr = opentype.Parse(antonTTF)
	})
	if fontErr != nil {
		return nil, fmt.Errorf("parse embedded font: %w", fontErr)
	}
	return anton, nil
}

func newFace(size float64) (font.Face, error) {
	f, err := loadFont()
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingNone,
	})
}

// RenderFile reads a template image and returns a captioned PNG.
func RenderFile(path, topText, bottomText, rawStyle string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode template: %w", err)
	}
	return Render(img, topText, bottomText, ParseStyle(rawStyle))
}

// Render draws captions onto src and encodes the result as PNG.
func Render(src image.Image, topText, bottomText string, style Style) ([]byte, error) {
	canvas := scaleCanvas(src, maxRenderSide)
	if err := drawCaptions(canvas, topText, bottomText, style); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, canvas); err != nil {
		return nil, fmt.Errorf("encode meme: %w", err)
	}
	return buf.Bytes(), nil
}

func scaleCanvas(src image.Image, maxSide int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	scale := math.Min(1, float64(maxSide)/float64(max(w, h)))
	dw, dh := int(math.Round(float64(w)*scale)), int(math.Round(float64(h)*scale))
	if dw < 1 {
		dw = 1
	}
	if dh < 1 {
		dh = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	if dw == w && dh == h {
		draw.Draw(dst, dst.Bounds(), src, b.Min, draw.Src)
	} else {
		draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	}
	return dst
}

func drawCaptions(canvas *image.RGBA, topText, bottomText string, style Style) error {
	upper := func(s string) string {
		if style.Uppercase {
			return strings.ToUpper(s)
		}
		return s
	}
	top := strings.TrimSpace(upper(topText))
	bottom := strings.TrimSpace(upper(bottomText))
	if top == "" && bottom == "" {
		return nil
	}

	width := float64(canvas.Bounds().Dx())
	height := float64(canvas.Bounds().Dy())
	maxWidth := width * 0.92
	base := width * 0.095 * style.FontScale
	minSize := math.Max(12, width*0.032)
	lineHeightRatio := 1.08

	size := math.Max(minSize, base)
	var face font.Face
	var topLines, bottomLines []string
	for guard := 0; guard < 40; guard++ {
		var err error
		face, err = newFace(size)
		if err != nil {
			return err
		}
		topLines = wrapText(top, face, maxWidth)
		bottomLines = wrapText(bottom, face, maxWidth)
		widest := 0.0
		for _, line := range append(append([]string{}, topLines...), bottomLines...) {
			if w := measure(face, line); w > widest {
				widest = w
			}
		}
		topHeight := float64(len(topLines)) * size * lineHeightRatio
		bottomHeight := float64(len(bottomLines)) * size * lineHeightRatio
		if widest <= width*0.94 && topHeight <= height*0.42 && bottomHeight <= height*0.42 {
			break
		}
		if size <= minSize {
			break
		}
		size = math.Max(minSize, size-math.Max(1, math.Round(size*0.055)))
	}
	if face == nil {
		return fmt.Errorf("could not lay out text")
	}
	if closer, ok := face.(interface{ Close() error }); ok {
		defer closer.Close()
	}

	lineHeight := size * 1.08
	fill := color.NRGBA(parseColor(style.Color))
	stroke := color.NRGBA(parseColor(style.Stroke))
	pad := height * 0.028
	// Canvas positions text from the line box top, while Go's font ascent is
	// much taller for Anton. This offset keeps server renders aligned with the
	// live editor preview.
	baselineOffset := size * 0.78

	topBase := pad + baselineOffset
	bottomBase := height - pad - float64(len(bottomLines))*lineHeight + baselineOffset
	if err := drawBlock(canvas, face, topLines, width/2, topBase, lineHeight, fill, stroke, style.Outline, size); err != nil {
		return err
	}
	return drawBlock(canvas, face, bottomLines, width/2, bottomBase, lineHeight, fill, stroke, style.Outline, size)
}

func drawBlock(
	canvas *image.RGBA,
	face font.Face,
	lines []string,
	centerX, firstBaseline, lineHeight float64,
	fill, stroke color.NRGBA,
	outline bool,
	size float64,
) error {
	strokeSrc := image.NewUniform(stroke)
	fillSrc := image.NewUniform(fill)
	radius := math.Max(1, size*0.06)
	const directions = 16

	for i, line := range lines {
		if line == "" {
			continue
		}
		x := centerX - measure(face, line)/2
		y := firstBaseline + float64(i)*lineHeight
		if outline && stroke.A > 0 {
			for d := 0; d < directions; d++ {
				angle := 2 * math.Pi * float64(d) / directions
				dx := math.Cos(angle) * radius
				dy := math.Sin(angle) * radius
				drawer := font.Drawer{
					Dst:  canvas,
					Src:  strokeSrc,
					Face: face,
					Dot:  fixed.P(int(math.Round(x+dx)), int(math.Round(y+dy))),
				}
				drawer.DrawString(line)
			}
		}
		drawer := font.Drawer{
			Dst:  canvas,
			Src:  fillSrc,
			Face: face,
			Dot:  fixed.P(int(math.Round(x)), int(math.Round(y))),
		}
		drawer.DrawString(line)
	}
	return nil
}

func wrapText(text string, face font.Face, maxWidth float64) []string {
	if text == "" {
		return nil
	}
	var lines []string
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			candidate := line + " " + word
			if measure(face, candidate) <= maxWidth {
				line = candidate
			} else {
				lines = append(lines, line)
				line = word
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func measure(face font.Face, text string) float64 {
	return float64(font.MeasureString(face, text)) / 64
}

func validHex(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for _, c := range value[1:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func parseColor(value string) color.RGBA {
	if !validHex(value) {
		return color.RGBA{R: 255, G: 255, B: 255, A: 255}
	}
	var r, g, b uint8
	fmt.Sscanf(value[1:], "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}
