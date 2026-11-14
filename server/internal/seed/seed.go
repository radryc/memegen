// Package seed fills a fresh database with meme templates from the Imgflip
// public API, plus a set of generated fallbacks when the network is down.
package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "image/gif"
	_ "image/jpeg"

	_ "golang.org/x/image/webp"

	"github.com/radryc/memebank/server/internal/store"
)

const (
	imgflipURL = "https://api.imgflip.com/get_memes"
	maxSeed    = 72
	maxBytes   = 15 << 20
)

// templateNameOverrides improve search aliases for common templates.
var templateNameOverrides = map[string]string{
	"129242436": "Change My Mind (Convince Me)",
}

func templateDeleted(st *store.Store, id string) bool {
	deleted, err := st.IsTemplateDeleted(id)
	return err == nil && deleted
}

type imgflipResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Memes []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"memes"`
	} `json:"data"`
}

// Ensure seeds templates if the database has none. Failures are non-fatal:
// the board still works with user uploads and generated fallbacks.
func Ensure(ctx context.Context, st *store.Store, uploadsDir string, log *slog.Logger, enabled bool) error {
	count, err := st.CountTemplates()
	if err != nil {
		return fmt.Errorf("count templates: %w", err)
	}
	dir := filepath.Join(uploadsDir, "templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create template dir: %w", err)
	}
	if count > 0 {
		ensureCuratedTemplates(ctx, st, dir, log)
		if err := ensureAnimatedTemplates(ctx, st, dir, log); err != nil {
			log.Warn("animated template seeding failed", "error", err)
		}
		backfillTemplateFingerprints(st, uploadsDir, log)
		return nil
	}
	if !enabled {
		log.Info("template seeding disabled")
		return nil
	}

	log.Info("seeding meme templates from imgflip", "max", maxSeed)
	inserted, err := seedImgflip(ctx, st, dir, log)
	if err != nil {
		log.Warn("imgflip seed failed, using generated templates", "error", err)
	}
	if inserted == 0 {
		n, ferr := seedFallbacks(st, dir)
		if ferr != nil {
			return fmt.Errorf("fallback seed: %w", ferr)
		}
		log.Info("generated fallback templates", "count", n)
		ensureCuratedTemplates(ctx, st, dir, log)
		if err := ensureAnimatedTemplates(ctx, st, dir, log); err != nil {
			log.Warn("animated template seeding failed", "error", err)
		}
		backfillTemplateFingerprints(st, uploadsDir, log)
		return nil
	}
	log.Info("seeded templates", "count", inserted)
	ensureCuratedTemplates(ctx, st, dir, log)
	if err := ensureAnimatedTemplates(ctx, st, dir, log); err != nil {
		log.Warn("animated template seeding failed", "error", err)
	}
	backfillTemplateFingerprints(st, uploadsDir, log)
	return nil
}

type download struct {
	id     string
	name   string
	url    string
	width  int
	height int
	index  int
}

func seedImgflip(ctx context.Context, st *store.Store, dir string, log *slog.Logger) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, imgflipURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "github.com/radryc/memebank/server/1.0 (meme template seeder)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("imgflip status %d", resp.StatusCode)
	}

	var payload imgflipResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&payload); err != nil {
		return 0, err
	}
	if !payload.Success {
		return 0, errors.New("imgflip returned success=false")
	}

	jobs := make([]download, 0, maxSeed)
	for i, m := range payload.Data.Memes {
		if len(jobs) >= maxSeed {
			break
		}
		if m.ID == "" || m.Name == "" || !strings.HasPrefix(m.URL, "https://") {
			continue
		}
		jobs = append(jobs, download{id: m.ID, name: m.Name, url: m.URL, width: m.Width, height: m.Height, index: i})
	}
	if len(jobs) == 0 {
		return 0, errors.New("imgflip returned no usable templates")
	}

	type result struct {
		job    download
		file   string
		width  int
		height int
		err    error
	}
	results := make(chan result, len(jobs))
	sem := make(chan struct{}, 6)
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		go func(job download) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			name, w, h, err := fetchTemplate(ctx, dir, job)
			results <- result{job: job, file: name, width: w, height: h, err: err}
		}(job)
	}
	wg.Wait()
	close(results)

	var collected []result
	for r := range results {
		if r.err != nil {
			log.Debug("template download failed", "name", r.job.name, "error", r.err)
			continue
		}
		collected = append(collected, r)
	}
	// Preserve the API's popularity order when inserting.
	for i := 0; i < len(collected); i++ {
		for j := i + 1; j < len(collected); j++ {
			if collected[j].job.index < collected[i].job.index {
				collected[i], collected[j] = collected[j], collected[i]
			}
		}
	}

	inserted := 0
	for _, r := range collected {
		if templateDeleted(st, r.job.id) {
			continue
		}
		name := r.job.name
		if override, ok := templateNameOverrides[r.job.id]; ok {
			name = override
		}
		t := store.Template{
			ID:        r.job.id,
			Name:      name,
			ImageURL:  "/uploads/templates/" + r.file,
			Width:     r.width,
			Height:    r.height,
			Source:    "imgflip",
			CreatedAt: time.Now().UTC().Add(time.Duration(inserted) * time.Millisecond),
		}
		if err := st.CreateTemplate(t); err != nil {
			log.Debug("insert template failed", "name", r.job.name, "error", err)
			continue
		}
		inserted++
	}
	if inserted == 0 {
		return 0, errors.New("no templates could be downloaded")
	}
	return inserted, nil
}

func fetchTemplate(ctx context.Context, dir string, job download) (string, int, int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, job.url, nil)
	if err != nil {
		return "", 0, 0, err
	}
	req.Header.Set("User-Agent", "github.com/radryc/memebank/server/1.0 (meme template seeder)")
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
		return "", 0, 0, errors.New("image too large")
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return "", 0, 0, fmt.Errorf("decode: %w", err)
	}
	if cfg.Width < 64 || cfg.Height < 64 || cfg.Width > 8000 || cfg.Height > 8000 {
		return "", 0, 0, fmt.Errorf("unsupported dimensions %dx%d", cfg.Width, cfg.Height)
	}
	ext := "." + format
	if format == "jpeg" {
		ext = ".jpg"
	}
	file := job.id + ext
	path := filepath.Join(dir, file)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", 0, 0, err
	}
	w := job.width
	h := job.height
	if w <= 0 || h <= 0 {
		w, h = cfg.Width, cfg.Height
	}
	return file, w, h, nil
}

// fallbackTheme describes a generated abstract template.
type fallbackTheme struct {
	name string
	from color.NRGBA
	to   color.NRGBA
	glow color.NRGBA
}

var fallbackThemes = []fallbackTheme{
	{"Blank Canvas · Acid", color.NRGBA{0x16, 0x2b, 0x12, 0xff}, color.NRGBA{0x0a, 0x0d, 0x08, 0xff}, color.NRGBA{0xc6, 0xff, 0x3d, 0xff}},
	{"Blank Canvas · Bubblegum", color.NRGBA{0x40, 0x0d, 0x2a, 0xff}, color.NRGBA{0x0b, 0x08, 0x0d, 0xff}, color.NRGBA{0xff, 0x3d, 0x8b, 0xff}},
	{"Blank Canvas · Midnight", color.NRGBA{0x13, 0x18, 0x40, 0xff}, color.NRGBA{0x07, 0x08, 0x12, 0xff}, color.NRGBA{0x7c, 0x6b, 0xff, 0xff}},
	{"Blank Canvas · Sunset", color.NRGBA{0x45, 0x18, 0x08, 0xff}, color.NRGBA{0x0d, 0x08, 0x06, 0xff}, color.NRGBA{0xff, 0x9d, 0x3d, 0xff}},
	{"Blank Canvas · Ice", color.NRGBA{0x08, 0x2b, 0x33, 0xff}, color.NRGBA{0x05, 0x0a, 0x0d, 0xff}, color.NRGBA{0x3d, 0xe8, 0xff, 0xff}},
	{"Blank Canvas · Toxic", color.NRGBA{0x1f, 0x33, 0x08, 0xff}, color.NRGBA{0x08, 0x0d, 0x05, 0xff}, color.NRGBA{0x9d, 0xff, 0x3d, 0xff}},
}

func seedFallbacks(st *store.Store, dir string) (int, error) {
	inserted := 0
	for i, theme := range fallbackThemes {
		id := fmt.Sprintf("canvas-%d", i+1)
		if templateDeleted(st, id) {
			continue
		}
		file := id + ".png"
		path := filepath.Join(dir, file)
		if err := writeFallback(path, theme); err != nil {
			return inserted, err
		}
		t := store.Template{
			ID:        id,
			Name:      theme.name,
			ImageURL:  "/uploads/templates/" + file,
			Width:     1200,
			Height:    900,
			Source:    "generated",
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Millisecond),
		}
		if err := st.CreateTemplate(t); err != nil {
			return inserted, err
		}
		inserted++
	}
	return inserted, nil
}

func writeFallback(path string, theme fallbackTheme) error {
	const w, h = 1200, 900
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: theme.to}, image.Point{}, draw.Src)

	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		row := color.NRGBA{
			R: lerp(theme.from.R, theme.to.R, t),
			G: lerp(theme.from.G, theme.to.G, t),
			B: lerp(theme.from.B, theme.to.B, t),
			A: 255,
		}
		draw.Draw(img, image.Rect(0, y, w, y+1), &image.Uniform{C: row}, image.Point{}, draw.Src)
	}
	// Two large soft blobs give the blank canvases a bit of depth.
	blobs := []struct {
		cx, cy, r float64
		alpha     float64
	}{
		{w * 0.78, h * 0.22, w * 0.34, 0.30},
		{w * 0.18, h * 0.85, w * 0.30, 0.22},
	}
	for _, b := range blobs {
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				dx := float64(x) - b.cx
				dy := float64(y) - b.cy
				d := math.Sqrt(dx*dx + dy*dy)
				if d >= b.r {
					continue
				}
				falloff := 1 - d/b.r
				a := falloff * falloff * b.alpha
				base := img.RGBAAt(x, y)
				img.SetRGBA(x, y, color.RGBA{
					R: lerp(base.R, theme.glow.R, a),
					G: lerp(base.G, theme.glow.G, a),
					B: lerp(base.B, theme.glow.B, a),
					A: 255,
				})
			}
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func lerp(a, b uint8, t float64) uint8 {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}
