package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/radryc/memebank/server/internal/store"
)

const (
	gifWidth  = 960
	gifHeight = 720
	gifFrames = 24
	gifDelay  = 6 // hundredths of a second
)

// ensureAnimatedTemplates adds GIF templates when the library has none.
// GIPHY_API_KEY can be set to pull trending meme GIFs; otherwise a set of
// generated animated canvases is created.
func ensureAnimatedTemplates(ctx context.Context, st *store.Store, dir string, log *slog.Logger) error {
	n, err := st.CountAnimatedTemplates()
	if err != nil {
		return fmt.Errorf("count gif templates: %w", err)
	}
	if n > 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if key := strings.TrimSpace(os.Getenv("GIPHY_API_KEY")); key != "" {
		inserted, gerr := seedGiphy(ctx, st, dir, key, log)
		if gerr != nil {
			log.Warn("giphy seed failed, generating animated canvases", "error", gerr)
		} else if inserted > 0 {
			log.Info("seeded gif templates from giphy", "count", inserted)
			return nil
		}
	}

	inserted, err := seedGeneratedGIFs(st, dir)
	if err != nil {
		return err
	}
	log.Info("generated animated gif templates", "count", inserted)
	return nil
}

func seedGeneratedGIFs(st *store.Store, dir string) (int, error) {
	themes := []struct {
		name    string
		pal     color.Palette
		painter func(*image.Paletted, int)
	}{
		{
			name: "GIF Canvas · Pulse",
			pal: color.Palette{
				color.RGBA{0x08, 0x0b, 0x06, 0xff},
				color.RGBA{0x1c, 0x2c, 0x0d, 0xff},
				color.RGBA{0x3d, 0x5e, 0x12, 0xff},
				color.RGBA{0x88, 0xc7, 0x27, 0xff},
				color.RGBA{0xc8, 0xff, 0x4d, 0xff},
				color.RGBA{0xff, 0xff, 0xff, 0xff},
			},
			painter: paintPulse,
		},
		{
			name: "GIF Canvas · Rave",
			pal: color.Palette{
				color.RGBA{0x0a, 0x06, 0x12, 0xff},
				color.RGBA{0xff, 0x2d, 0x7f, 0xff},
				color.RGBA{0x7c, 0x5c, 0xff, 0xff},
				color.RGBA{0x5c, 0xe1, 0xff, 0xff},
				color.RGBA{0xc8, 0xff, 0x4d, 0xff},
				color.RGBA{0xff, 0xff, 0xff, 0xff},
			},
			painter: paintRave,
		},
		{
			name: "GIF Canvas · Confetti",
			pal: color.Palette{
				color.RGBA{0x07, 0x07, 0x0a, 0xff},
				color.RGBA{0xc8, 0xff, 0x4d, 0xff},
				color.RGBA{0xff, 0x2d, 0x7f, 0xff},
				color.RGBA{0x5c, 0xe1, 0xff, 0xff},
				color.RGBA{0xff, 0xd1, 0x66, 0xff},
				color.RGBA{0xff, 0xff, 0xff, 0xff},
			},
			painter: paintConfetti,
		},
		{
			name: "GIF Canvas · Orbit",
			pal: color.Palette{
				color.RGBA{0x06, 0x07, 0x0f, 0xff},
				color.RGBA{0x1a, 0x1b, 0x33, 0xff},
				color.RGBA{0xc8, 0xff, 0x4d, 0xff},
				color.RGBA{0xff, 0x2d, 0x7f, 0xff},
				color.RGBA{0x5c, 0xe1, 0xff, 0xff},
				color.RGBA{0xff, 0xff, 0xff, 0xff},
			},
			painter: paintOrbit,
		},
		{
			name: "GIF Canvas · Warp",
			pal: color.Palette{
				color.RGBA{0x04, 0x07, 0x10, 0xff},
				color.RGBA{0x0c, 0x2a, 0x3a, 0xff},
				color.RGBA{0x13, 0x5a, 0x6e, 0xff},
				color.RGBA{0x1d, 0x8f, 0x9f, 0xff},
				color.RGBA{0x5c, 0xe1, 0xff, 0xff},
				color.RGBA{0xff, 0xff, 0xff, 0xff},
			},
			painter: paintWarp,
		},
	}

	inserted := 0
	for i, theme := range themes {
		id := fmt.Sprintf("gif-canvas-%d", i+1)
		if templateDeleted(st, id) {
			continue
		}
		file := id + ".gif"
		path := filepath.Join(dir, file)
		if err := writeAnimatedTemplate(path, theme.pal, theme.painter); err != nil {
			return inserted, err
		}
		t := store.Template{
			ID:        id,
			Name:      theme.name,
			ImageURL:  "/uploads/templates/" + file,
			Width:     gifWidth,
			Height:    gifHeight,
			Source:    "generated",
			Animated:  true,
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Millisecond),
		}
		if err := st.CreateTemplate(t); err != nil {
			return inserted, err
		}
		inserted++
	}
	return inserted, nil
}

func writeAnimatedTemplate(path string, pal color.Palette, paint func(*image.Paletted, int)) error {
	frames := make([]*image.Paletted, 0, gifFrames)
	delays := make([]int, 0, gifFrames)
	for i := 0; i < gifFrames; i++ {
		img := image.NewPaletted(image.Rect(0, 0, gifWidth, gifHeight), pal)
		paint(img, i)
		frames = append(frames, img)
		delays = append(delays, gifDelay)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.EncodeAll(f, &gif.GIF{Image: frames, Delay: delays, LoopCount: 0})
}

func fillPaletted(img *image.Paletted, index uint8) {
	for i := range img.Pix {
		img.Pix[i] = index
	}
}

func fillRect(img *image.Paletted, x, y, w, h float64, index uint8) {
	b := img.Bounds()
	x0, y0 := int(x), int(y)
	x1, y1 := int(x+w), int(y+h)
	for py := max(y0, b.Min.Y); py < min(y1, b.Max.Y); py++ {
		row := py * img.Stride
		for px := max(x0, b.Min.X); px < min(x1, b.Max.X); px++ {
			img.Pix[row+px] = index
		}
	}
}

func fillCircle(img *image.Paletted, cx, cy, r float64, index uint8) {
	if r <= 0 {
		return
	}
	b := img.Bounds()
	x0, x1 := int(cx-r), int(cx+r)
	y0, y1 := int(cy-r), int(cy+r)
	r2 := r * r
	for py := max(y0, b.Min.Y); py < min(y1, b.Max.Y); py++ {
		dy := float64(py) - cy
		row := py * img.Stride
		for px := max(x0, b.Min.X); px < min(x1, b.Max.X); px++ {
			dx := float64(px) - cx
			if dx*dx+dy*dy <= r2 {
				img.Pix[row+px] = index
			}
		}
	}
}

func ring(img *image.Paletted, cx, cy, r, thickness float64, index uint8) {
	if r <= 0 {
		return
	}
	b := img.Bounds()
	x0, x1 := int(cx-r-thickness), int(cx+r+thickness)
	y0, y1 := int(cy-r-thickness), int(cy+r+thickness)
	inner := (r - thickness) * (r - thickness)
	outer := (r + thickness) * (r + thickness)
	for py := max(y0, b.Min.Y); py < min(y1, b.Max.Y); py++ {
		dy := float64(py) - cy
		row := py * img.Stride
		for px := max(x0, b.Min.X); px < min(x1, b.Max.X); px++ {
			dx := float64(px) - cx
			d2 := dx*dx + dy*dy
			if d2 >= inner && d2 <= outer {
				img.Pix[row+px] = index
			}
		}
	}
}

func paintPulse(img *image.Paletted, frame int) {
	fillPaletted(img, 0)
	phase := 2 * math.Pi * float64(frame) / gifFrames
	cx := float64(gifWidth)/2 + math.Cos(phase)*gifWidth*0.16
	cy := float64(gifHeight)/2 + math.Sin(phase)*gifHeight*0.12
	pulse := 1 + 0.09*math.Sin(phase*2)
	for i := 0; i < 4; i++ {
		r := (1 - float64(i)*0.21) * gifHeight * 0.33 * pulse
		fillCircle(img, cx, cy, r, uint8(1+i))
	}
	fillCircle(img, cx, cy, gifHeight*0.035, 5)

	// Two accent rings drifting in the opposite direction.
	for i := 0; i < 2; i++ {
		angle := -phase*0.8 + float64(i)*math.Pi
		ox := float64(gifWidth)/2 + math.Cos(angle)*gifWidth*0.35
		oy := float64(gifHeight)/2 + math.Sin(angle)*gifHeight*0.28
		fillCircle(img, ox, oy, gifHeight*0.025, 5)
	}
}

func paintRave(img *image.Paletted, frame int) {
	fillPaletted(img, 0)
	for x := 0; x < gifWidth; x += 48 {
		band := x / 48
		h := gifHeight * (0.22 + 0.18*math.Abs(math.Sin(float64(frame)/gifFrames*2*math.Pi+float64(band)*0.55)))
		ci := uint8(1 + (band+frame/3)%4)
		fillRect(img, float64(x), float64(gifHeight)-h, 32, h, ci)
	}
	y := float64(gifHeight) * (0.15 + 0.7*math.Abs(math.Sin(float64(frame)/gifFrames*math.Pi)))
	fillRect(img, 0, y, gifWidth, 4, 5)
}

func paintConfetti(img *image.Paletted, frame int) {
	fillPaletted(img, 0)
	for i := 0; i < 90; i++ {
		seed := float64(i) * 12.9898
		x := math.Mod(math.Abs(math.Sin(seed))*43758.5453, gifWidth)
		speed := 0.05 + math.Mod(math.Abs(math.Sin(seed*0.7))*43758.5453, 1)*0.14
		base := math.Mod(math.Abs(math.Sin(seed*0.3))*43758.5453, gifHeight)
		y := math.Mod(base+float64(frame)*speed*gifHeight, gifHeight+40) - 20
		w := 6 + math.Mod(seed*3, 9)
		h := 10 + math.Mod(seed*5, 16)
		ci := uint8(1 + i%4)
		fillRect(img, x, y, w, h, ci)
	}
}

func paintOrbit(img *image.Paletted, frame int) {
	fillPaletted(img, 0)
	phase := 2 * math.Pi * float64(frame) / gifFrames
	cx, cy := float64(gifWidth)/2, float64(gifHeight)/2
	for i := 1; i <= 3; i++ {
		ring(img, cx, cy, gifHeight*0.13*float64(i), 2, 1)
	}
	for i := 0; i < 3; i++ {
		angle := phase + float64(i)*2*math.Pi/3
		r := gifHeight * 0.13 * float64(i+1)
		x := cx + math.Cos(angle)*r
		y := cy + math.Sin(angle)*r
		fillCircle(img, x, y, 18+float64(i)*5, uint8(2+i))
	}
	fillCircle(img, cx, cy, 26, 5)
}

func paintWarp(img *image.Paletted, frame int) {
	offset := float64(frame) * 9
	for y := 0; y < gifHeight; y++ {
		row := y * img.Stride
		for x := 0; x < gifWidth; x++ {
			band := (float64(x) + float64(y)*0.55 + offset) / 70
			img.Pix[row+x] = uint8(1 + int(math.Abs(math.Sin(band*0.8))*4)%4)
		}
	}
	// A bright line shooting across the frame.
	y := math.Mod(float64(frame)*gifHeight/float64(gifFrames), gifHeight)
	fillRect(img, 0, y, gifWidth, 6, 4)
	ring(img, gifWidth/2, gifHeight/2, gifHeight*0.3, 3, 5)
}

// --- optional Giphy seeding ------------------------------------------------

type giphyResponse struct {
	Data []struct {
		Title  string `json:"title"`
		Images struct {
			Downsized struct {
				URL string `json:"url"`
			} `json:"downsized"`
		} `json:"images"`
	} `json:"data"`
}

func seedGiphy(ctx context.Context, st *store.Store, dir, key string, log *slog.Logger) (int, error) {
	url := "https://api.giphy.com/v1/gifs/search?q=meme&limit=12&rating=pg-13&api_key=" + key
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("giphy status %d", resp.StatusCode)
	}
	var payload giphyResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&payload); err != nil {
		return 0, err
	}

	inserted := 0
	for _, item := range payload.Data {
		gifURL := item.Images.Downsized.URL
		if !strings.HasPrefix(gifURL, "https://") || !strings.HasSuffix(strings.ToLower(gifURL), ".gif") {
			continue
		}
		name := strings.TrimSpace(item.Title)
		if name == "" {
			name = "Trending GIF"
		}
		if len(name) > 70 {
			name = name[:70]
		}
		file, width, height, err := downloadGIF(ctx, dir, gifURL)
		if err != nil {
			log.Debug("giphy gif download failed", "error", err)
			continue
		}
		t := store.Template{
			ID:        store.NewID(),
			Name:      name,
			ImageURL:  "/uploads/templates/" + file,
			Width:     width,
			Height:    height,
			Source:    "giphy",
			Animated:  true,
			CreatedAt: time.Now().UTC().Add(time.Duration(inserted) * time.Millisecond),
		}
		if err := st.CreateTemplate(t); err != nil {
			os.Remove(filepath.Join(dir, file))
			continue
		}
		inserted++
		if inserted >= 10 {
			break
		}
	}
	if inserted == 0 {
		return 0, errors.New("no giphy gifs could be downloaded")
	}
	return inserted, nil
}

func downloadGIF(ctx context.Context, dir, url string) (string, int, int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, 0, err
	}
	req.Header.Set("User-Agent", "github.com/radryc/memebank/server/1.0 (gif template seeder)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", 0, 0, fmt.Errorf("status %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return "", 0, 0, err
	}
	if len(data) > maxBytes {
		return "", 0, 0, errors.New("gif too large")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "gif" {
		return "", 0, 0, errors.New("not a gif")
	}
	if cfg.Width < 64 || cfg.Height < 64 || cfg.Width > 2000 || cfg.Height > 2000 {
		return "", 0, 0, errors.New("unsupported gif size")
	}
	file := store.NewID() + ".gif"
	if err := os.WriteFile(filepath.Join(dir, file), data, 0o644); err != nil {
		return "", 0, 0, err
	}
	return file, cfg.Width, cfg.Height, nil
}
