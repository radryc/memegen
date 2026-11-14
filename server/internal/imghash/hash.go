// Package imghash provides exact and perceptual fingerprints for images.
package imghash

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"math/bits"
	"os"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	// hashWidth/hashHeight drive a 256-bit difference hash (dHash). Resizing
	// to 17x16 and comparing horizontal neighbours is robust to re-encoding
	// and small resizes while staying cheap.
	hashWidth  = 17
	hashHeight = 16

	// MaxSimilarDistance is the dHash Hamming distance under which two images
	// are considered visually identical. Re-encoding or resizing typically
	// lands around 6-12 bits; distinct templates sit much higher.
	MaxSimilarDistance = 14
)

// Exact returns the SHA-256 fingerprint of raw image bytes.
func Exact(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// Perceptual returns a 256-bit dHash as hex. The image is converted to
// grayscale, resized to 17x16, and each row is reduced to 16 comparisons.
func Perceptual(img image.Image) string {
	small := image.NewGray(image.Rect(0, 0, hashWidth, hashHeight))
	draw.ApproxBiLinear.Scale(small, small.Bounds(), img, img.Bounds(), draw.Src, nil)

	out := make([]byte, (hashWidth-1)*hashHeight/8) // 32 bytes = 256 bits
	bit := 0
	for y := 0; y < hashHeight; y++ {
		for x := 0; x < hashWidth-1; x++ {
			if small.GrayAt(x, y).Y > small.GrayAt(x+1, y).Y {
				out[bit/8] |= 1 << (7 - uint(bit%8))
			}
			bit++
		}
	}
	return hex.EncodeToString(out)
}

// PerceptualFile decodes an image file (first frame for animated GIFs) and
// returns its perceptual hash.
func PerceptualFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}
	return Perceptual(img), nil
}

// Distance returns the Hamming distance between two hex perceptual hashes.
// Invalid or mismatched hashes return 256, which is treated as "not similar".
func Distance(a, b string) int {
	if a == "" || b == "" || len(a) != len(b) {
		return MaxSimilarDistance * 32
	}
	ab, err1 := hex.DecodeString(a)
	bb, err2 := hex.DecodeString(b)
	if err1 != nil || err2 != nil || len(ab) != len(bb) {
		return MaxSimilarDistance * 32
	}
	dist := 0
	for i := range ab {
		dist += bits.OnesCount8(ab[i] ^ bb[i])
	}
	return dist
}

// AspectRatio returns width/height (0 for invalid dimensions).
func AspectRatio(width, height int) float64 {
	if width <= 0 || height <= 0 {
		return 0
	}
	return float64(width) / float64(height)
}
