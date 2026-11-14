package render

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestRenderAddsCaptions(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 600, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 600; x++ {
			src.SetRGBA(x, y, color.RGBA{R: 20, G: 40, B: 80, A: 255})
		}
	}
	out, err := Render(src, "when the build passes", "on the first try", ParseStyle("{}"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	img, err := png.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	if img.Bounds().Dx() != 600 || img.Bounds().Dy() != 400 {
		t.Fatalf("size = %v", img.Bounds())
	}

	// Captions are drawn near the top and bottom edges; look for bright pixels
	// that were not part of the dark source image.
	brightTop, brightBottom := 0, 0
	for y := 0; y < 80; y++ {
		for x := 0; x < 600; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r > 0xff00 && g > 0xff00 && b > 0xff00 {
				brightTop++
			}
		}
	}
	for y := 320; y < 400; y++ {
		for x := 0; x < 600; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			if r > 0xff00 && g > 0xff00 && b > 0xff00 {
				brightBottom++
			}
		}
	}
	if brightTop == 0 || brightBottom == 0 {
		t.Fatalf("captions missing: top=%d bottom=%d", brightTop, brightBottom)
	}
}

func TestStyleDefaults(t *testing.T) {
	style := ParseStyle("")
	if style.FontScale != 1 || style.Color != "#ffffff" || !style.Uppercase || !style.Outline {
		t.Fatalf("defaults = %+v", style)
	}
	custom := ParseStyle(`{"fontScale":1.25,"color":"#c8ff4d","stroke":"#000000","uppercase":false,"outline":false}`)
	if custom.FontScale != 1.25 || custom.Color != "#c8ff4d" || custom.Uppercase || custom.Outline {
		t.Fatalf("custom = %+v", custom)
	}
	if got := ParseStyle(`{"color":"nope"}`).Color; got != "#ffffff" {
		t.Fatalf("invalid color = %q", got)
	}
}
