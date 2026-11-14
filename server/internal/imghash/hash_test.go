package imghash

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"

	"golang.org/x/image/draw"
)

func sampleImage(shapes bool) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 400, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 400; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(x * 255 / 400),
				G: uint8(y * 255 / 300),
				B: 60,
				A: 255,
			})
		}
	}
	box := image.Rect(80, 60, 240, 220)
	if shapes {
		box = image.Rect(220, 40, 360, 260)
	}
	draw.Draw(img, box, &image.Uniform{C: color.RGBA{250, 240, 30, 255}}, image.Point{}, draw.Src)
	return img
}

func TestPerceptualSurvivesReencode(t *testing.T) {
	img := sampleImage(true)
	orig := Perceptual(img)
	if len(orig) != 64 {
		t.Fatalf("hash length = %d", len(orig))
	}

	// Downscale, re-encode as JPEG, and decode again.
	small := image.NewRGBA(image.Rect(0, 0, 200, 150))
	draw.CatmullRom.Scale(small, small.Bounds(), img, img.Bounds(), draw.Src, nil)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, small, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	decoded, _, err := image.Decode(&buf)
	if err != nil {
		t.Fatal(err)
	}
	dist := Distance(orig, Perceptual(decoded))
	if dist > MaxSimilarDistance {
		t.Fatalf("re-encoded distance = %d, want <= %d", dist, MaxSimilarDistance)
	}
}

func TestPerceptualDistinguishesImages(t *testing.T) {
	a := Perceptual(sampleImage(true))
	b := Perceptual(sampleImage(false))
	if Distance(a, b) <= MaxSimilarDistance {
		t.Fatalf("different images look identical: distance %d", Distance(a, b))
	}
}

func TestDistanceInvalid(t *testing.T) {
	if got := Distance("", "abc"); got <= MaxSimilarDistance {
		t.Fatalf("invalid distance = %d", got)
	}
	if got := Distance("00", "ff"); got != 8 {
		t.Fatalf("distance 00/ff = %d", got)
	}
}
