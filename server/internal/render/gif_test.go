package render

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderGIFPreservesFrames(t *testing.T) {
	pal := color.Palette{
		color.RGBA{0, 0, 0, 255},
		color.RGBA{200, 40, 40, 255},
		color.RGBA{40, 160, 220, 255},
	}
	var frames []*image.Paletted
	for i := 0; i < 3; i++ {
		frame := image.NewPaletted(image.Rect(0, 0, 240, 160), pal)
		for y := 0; y < 160; y++ {
			for x := 0; x < 240; x++ {
				frame.SetColorIndex(x, y, 2)
			}
		}
		for y := 40; y < 120; y++ {
			for x := 20 + i*60; x < 100+i*60; x++ {
				frame.SetColorIndex(x, y, 1)
			}
		}
		frames = append(frames, frame)
	}
	var src bytes.Buffer
	if err := gif.EncodeAll(&src, &gif.GIF{Image: frames, Delay: []int{10, 10, 10}, LoopCount: 0}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "template.gif")
	if err := os.WriteFile(path, src.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := RenderGIFFile(path, "GIF MEME", "ON DEMAND", "{}")
	if err != nil {
		t.Fatalf("render gif: %v", err)
	}
	decoded, err := gif.DecodeAll(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if len(decoded.Image) != 3 {
		t.Fatalf("frames = %d", len(decoded.Image))
	}
	if decoded.Image[0].Bounds().Dx() != 240 || decoded.Image[0].Bounds().Dy() != 160 {
		t.Fatalf("size = %v", decoded.Image[0].Bounds())
	}
	if decoded.LoopCount != 0 {
		t.Fatalf("loop = %d", decoded.LoopCount)
	}
	if len(out) < 1000 {
		t.Fatalf("output too small: %d bytes", len(out))
	}
}

func TestRenderGIFManyFrames(t *testing.T) {
	pal := color.Palette{
		color.RGBA{10, 10, 20, 255},
		color.RGBA{200, 255, 77, 255},
	}
	var frames []*image.Paletted
	for i := 0; i < 30; i++ {
		frame := image.NewPaletted(image.Rect(0, 0, 96, 72), pal)
		for y := 0; y < 72; y++ {
			for x := 0; x < 96; x++ {
				frame.SetColorIndex(x, y, 0)
			}
		}
		for y := 20; y < 50; y++ {
			for x := i; x < i+20; x++ {
				frame.SetColorIndex(x, y, 1)
			}
		}
		frames = append(frames, frame)
	}
	var src bytes.Buffer
	if err := gif.EncodeAll(&src, &gif.GIF{Image: frames, Delay: make([]int, len(frames))}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "many.gif")
	if err := os.WriteFile(path, src.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := RenderGIFFile(path, "MANY FRAMES", "NO DEADLOCK", "{}")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	decoded, err := gif.DecodeAll(bytes.NewReader(out))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded.Image) != 30 {
		t.Fatalf("frames = %d", len(decoded.Image))
	}
}
