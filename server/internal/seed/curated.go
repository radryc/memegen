package seed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/radryc/memebank/server/internal/imghash"
	"github.com/radryc/memebank/server/internal/store"
)

// curatedTemplates are handpicked classics that the public Imgflip get_memes
// endpoint does not return. They are backfilled on startup by id.
type curatedTemplate struct {
	ID   string
	Name string
	URL  string
}

var curatedTemplates = []curatedTemplate{
	{
		ID:   "curated-facepalm",
		Name: "Facepalm",
		URL:  "https://i.imgflip.com/wczz.jpg",
	},
	{
		ID:   "curated-computer-guy-facepalm",
		Name: "Computer Guy Facepalm",
		URL:  "https://i.imgflip.com/cdl6.jpg",
	},
	{
		ID:   "curated-young-thug-computer",
		Name: "Young Thug at Computer",
		URL:  "https://i.imgflip.com/6wegou.jpg",
	},
	{
		ID:   "curated-first-world-problems",
		Name: "First World Problems",
		URL:  "https://i.imgflip.com/1bhf.jpg",
	},
	{
		ID:   "curated-elon-laughing",
		Name: "Elon Musk Laughing",
		URL:  "https://i.imgflip.com/4xfw0m.jpg",
	},
	{
		ID:   "curated-elon-confused",
		Name: "Confused Elon Musk",
		URL:  "https://i.imgflip.com/1vgnzi.jpg",
	},
	{
		ID:   "curated-elon-dancing",
		Name: "Elon Musk Dancing",
		URL:  "https://i.imgflip.com/249qqg.jpg",
	},
}

// ensureCuratedTemplates adds any missing handpicked templates. It is
// idempotent, so existing boards pick them up on the next restart.
func ensureCuratedTemplates(ctx context.Context, st *store.Store, dir string, log *slog.Logger) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Warn("could not create template dir", "error", err)
		return
	}
	// Backdate these so the handpicked set leads the library rather than being
	// buried at the end of the seeded classics.
	base := time.Now().UTC().Add(-1000 * time.Hour)

	inserted := 0
	for i, item := range curatedTemplates {
		if _, err := st.GetTemplate(item.ID); err == nil {
			continue
		} else if !errors.Is(err, store.ErrNotFound) {
			log.Warn("could not check curated template", "name", item.Name, "error", err)
			continue
		}
		if templateDeleted(st, item.ID) {
			continue
		}

		file, width, height, err := downloadCurated(ctx, dir, item.ID, item.URL)
		if err != nil {
			log.Warn("could not download curated template", "name", item.Name, "error", err)
			continue
		}
		t := store.Template{
			ID:        item.ID,
			Name:      item.Name,
			ImageURL:  "/uploads/templates/" + file,
			Width:     width,
			Height:    height,
			Source:    "imgflip",
			CreatedAt: base.Add(time.Duration(i) * time.Millisecond),
		}
		if err := st.CreateTemplate(t); err != nil {
			os.Remove(filepath.Join(dir, file))
			log.Warn("could not insert curated template", "name", item.Name, "error", err)
			continue
		}
		inserted++
	}
	if inserted > 0 {
		log.Info("added handpicked templates", "count", inserted)
	}
}

func downloadCurated(ctx context.Context, dir, id, url string) (string, int, int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return "", 0, 0, err
	}
	req.Header.Set("User-Agent", "github.com/radryc/memebank/server/1.0 (curated template seeder)")
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
	file := id + ext
	if err := os.WriteFile(filepath.Join(dir, file), data, 0o644); err != nil {
		return "", 0, 0, err
	}
	return file, cfg.Width, cfg.Height, nil
}

// backfillTemplateFingerprints fingerprints template files that predate upload
// dedupe support, so re-uploading an existing image is detected immediately.
func backfillTemplateFingerprints(st *store.Store, uploadsDir string, log *slog.Logger) {
	items, err := st.ListTemplatesMissingFingerprints()
	if err != nil {
		log.Warn("could not list templates for hashing", "error", err)
		return
	}
	updated := 0
	for _, item := range items {
		if !strings.HasPrefix(item.ImageURL, "/uploads/") {
			continue
		}
		rel := strings.TrimPrefix(item.ImageURL, "/uploads/")
		path := filepath.Join(uploadsDir, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			log.Debug("could not read template for hashing", "id", item.ID, "error", err)
			continue
		}
		exact := item.ContentHash
		if exact == "" {
			sum := sha256.Sum256(data)
			exact = hex.EncodeToString(sum[:])
		}
		perceptual := item.PerceptualHash
		if perceptual == "" {
			perceptual, err = imghash.PerceptualFile(path)
			if err != nil {
				log.Debug("could not hash template visually", "id", item.ID, "error", err)
				// Still store the exact hash so it is usable; the perceptual
				// hash will be retried on the next start.
				if setErr := st.SetTemplateFingerprints(item.ID, exact, ""); setErr != nil {
					log.Warn("could not store template hash", "id", item.ID, "error", setErr)
				}
				continue
			}
		}
		if err := st.SetTemplateFingerprints(item.ID, exact, perceptual); err != nil {
			log.Warn("could not store template fingerprints", "id", item.ID, "error", err)
			continue
		}
		updated++
	}
	if updated > 0 {
		log.Info("fingerprinted templates", "count", updated)
	}
}
