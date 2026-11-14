// Package api exposes the memebank HTTP API and serves the built frontend.
package api

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"

	"github.com/radryc/memebank/server/internal/imghash"
	"github.com/radryc/memebank/server/internal/render"
	"github.com/radryc/memebank/server/internal/store"
)

const maxUploadBytes = 12 << 20

var voterIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,64}$`)

// Server holds dependencies shared by the HTTP handlers.
type Server struct {
	st         *store.Store
	uploads    string
	staticDir  string
	log        *slog.Logger
	mux        *http.ServeMux
	templateMu sync.Mutex
	imageGroup singleflight.Group
	imageSem   chan struct{}
	imageCache *render.ByteCache
}

// New builds the HTTP handler for the API, uploads and the SPA.
func New(st *store.Store, uploadsDir, staticDir string, log *slog.Logger) http.Handler {
	s := &Server{
		st:         st,
		uploads:    uploadsDir,
		staticDir:  staticDir,
		log:        log,
		mux:        http.NewServeMux(),
		imageSem:   make(chan struct{}, 2),
		imageCache: render.NewByteCache(64 << 20),
	}

	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/stats", s.handleStats)
	s.mux.HandleFunc("GET /api/templates", s.handleTemplates)
	s.mux.HandleFunc("POST /api/templates", s.handleCreateTemplate)
	s.mux.HandleFunc("GET /api/memes", s.handleMemes)
	s.mux.HandleFunc("POST /api/memes", s.handleCreateMeme)
	s.mux.HandleFunc("GET /api/memes/{id}", s.handleMeme)
	s.mux.HandleFunc("GET /api/memes/{id}/image", s.handleMemeImage)
	s.mux.HandleFunc("POST /api/memes/{id}/vote", s.handleVote)
	s.mux.HandleFunc("GET /api/leaderboard", s.handleLeaderboard)

	uploads := http.StripPrefix("/uploads/", http.FileServer(http.Dir(filepath.Join(uploadsDir))))
	s.mux.Handle("GET /uploads/", cacheControl(uploads))

	if _, err := os.Stat(filepath.Join(staticDir, "index.html")); err == nil {
		s.mux.Handle("/", spaHandler(staticDir))
	} else {
		log.Warn("built frontend not found; serving API only", "static", staticDir)
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				writeJSON(w, http.StatusOK, map[string]string{
					"service": "memebank api",
					"hint":    "run `npm run build` in web/ (or use the vite dev server) to serve the UI",
				})
				return
			}
			writeErr(w, http.StatusNotFound, "not found")
		})
	}

	return s.recoverer(s.cors(s.securityHeaders(s.logRequests(s.mux))))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	st, err := s.st.Stats()
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not load stats", err)
		return
	}
	writeJSON(w, http.StatusOK, st)
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	media := r.URL.Query().Get("media")
	templates, err := s.st.ListTemplates(source, search, media)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not load templates", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"templates": templates, "total": len(templates)})
}

func (s *Server) handleCreateTemplate(w http.ResponseWriter, r *http.Request) {
	if err := parseMultipart(w, r); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		name = "Community upload"
	}
	if len(name) > 70 {
		name = name[:70]
	}

	file, _, err := r.FormFile("image")
	if err != nil {
		writeErr(w, http.StatusBadRequest, "an image file is required")
		return
	}
	defer file.Close()

	data, cfg, format, err := readImage(file)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if cfg.Width < 64 || cfg.Height < 64 || cfg.Width > 8000 || cfg.Height > 8000 {
		writeErr(w, http.StatusBadRequest, "image must be between 64 and 8000 pixels per side")
		return
	}

	// Fingerprint the bytes so the same image is never uploaded twice. The
	// mutex closes the check-then-insert window for concurrent uploads.
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	s.templateMu.Lock()
	defer s.templateMu.Unlock()

	duplicate := false
	match := ""
	var existing store.Template
	existing, err = s.st.GetTemplateByHash(hash)
	if err == nil {
		duplicate = true
		match = "exact"
	} else if !errors.Is(err, store.ErrNotFound) {
		s.fail(w, http.StatusInternalServerError, "could not check for duplicates", err)
		return
	}

	perceptual := ""
	if !duplicate && cfg.Width*cfg.Height <= 25_000_000 {
		if img, _, decodeErr := image.Decode(bytes.NewReader(data)); decodeErr == nil {
			perceptual = imghash.Perceptual(img)
			aspect := float64(cfg.Width) / float64(cfg.Height)
			similar, similarErr := s.st.FindSimilarTemplate(perceptual, aspect, format == "gif")
			if similarErr == nil {
				existing = similar
				duplicate = true
				match = "similar"
			} else if !errors.Is(similarErr, store.ErrNotFound) {
				s.log.Warn("perceptual duplicate check failed", "error", similarErr)
			}
		}
	}

	if duplicate {
		writeJSON(w, http.StatusOK, map[string]any{
			"template":  existing,
			"duplicate": true,
			"match":     match,
		})
		return
	}

	filename, err := saveUpload(filepath.Join(s.uploads, "templates"), data, extensionFor(format))
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not save image", err)
		return
	}

	t := store.Template{
		ID:             store.NewID(),
		Name:           name,
		ImageURL:       "/uploads/templates/" + filename,
		Width:          cfg.Width,
		Height:         cfg.Height,
		Source:         "upload",
		Animated:       store.IsAnimatedURL("/uploads/templates/" + filename),
		ContentHash:    hash,
		PerceptualHash: perceptual,
		CreatedAt:      time.Now().UTC(),
	}
	if err := s.st.CreateTemplate(t); err != nil {
		s.fail(w, http.StatusInternalServerError, "could not save template", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"template": t})
}

func (s *Server) handleMemes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	opts := store.ListOptions{
		Sort:     q.Get("sort"),
		Search:   strings.TrimSpace(q.Get("q")),
		Template: q.Get("template"),
		VoterID:  voterID(r),
	}
	opts.Limit = intParam(q.Get("limit"), 24)
	opts.Offset = intParam(q.Get("offset"), 0)

	memes, hasMore, err := s.st.ListMemes(opts)
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not load memes", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"memes":      memes,
		"hasMore":    hasMore,
		"nextOffset": opts.Offset + len(memes),
	})
}

func (s *Server) handleMeme(w http.ResponseWriter, r *http.Request) {
	m, err := s.st.GetMeme(r.PathValue("id"), voterID(r))
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "meme not found")
		return
	}
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not load meme", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"meme": m})
}

func (s *Server) handleCreateMeme(w http.ResponseWriter, r *http.Request) {
	if err := parseMultipart(w, r); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	if len(title) > 120 {
		title = title[:120]
	}
	author := strings.TrimSpace(r.FormValue("author"))
	if author == "" {
		author = "anon"
	}
	if len(author) > 40 {
		author = author[:40]
	}
	top := strings.TrimSpace(r.FormValue("topText"))
	bottom := strings.TrimSpace(r.FormValue("bottomText"))
	if len(top) > 280 {
		top = top[:280]
	}
	if len(bottom) > 280 {
		bottom = bottom[:280]
	}
	style := strings.TrimSpace(r.FormValue("style"))
	if style == "" {
		style = "{}"
	}
	if len(style) > 2048 || !json.Valid([]byte(style)) {
		writeErr(w, http.StatusBadRequest, "style must be valid JSON under 2KB")
		return
	}

	templateID := strings.TrimSpace(r.FormValue("templateId"))
	if templateID != "" {
		if _, err := s.st.GetTemplate(templateID); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusBadRequest, "unknown templateId")
			return
		} else if err != nil {
			s.fail(w, http.StatusInternalServerError, "could not check template", err)
			return
		}
	}

	// New static memes are stored as a spec only. An image is still accepted
	// (and required for animated templates) for backwards compatibility.
	imageURL := ""
	file, _, fileErr := r.FormFile("image")
	switch {
	case fileErr == nil:
		defer file.Close()
		data, cfg, format, err := readImage(file)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if cfg.Width < 64 || cfg.Height < 64 || cfg.Width > 10000 || cfg.Height > 10000 {
			writeErr(w, http.StatusBadRequest, "meme image must be between 64 and 10000 pixels per side")
			return
		}
		filename, err := saveUpload(filepath.Join(s.uploads, "memes"), data, extensionFor(format))
		if err != nil {
			s.fail(w, http.StatusInternalServerError, "could not save meme", err)
			return
		}
		imageURL = "/uploads/memes/" + filename
	case errors.Is(fileErr, http.ErrMissingFile):
		if templateID == "" {
			writeErr(w, http.StatusBadRequest, "choose a template before publishing")
			return
		}
	default:
		writeErr(w, http.StatusBadRequest, "could not read the meme image")
		return
	}

	m, err := s.st.CreateMeme(store.Meme{
		ID:         store.NewID(),
		Title:      title,
		Author:     author,
		ImageURL:   imageURL,
		TemplateID: templateID,
		TopText:    top,
		BottomText: bottom,
		Style:      style,
		CreatedAt:  time.Now().UTC(),
	})
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not publish meme", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"meme": m})
}

func (s *Server) handleMemeImage(w http.ResponseWriter, r *http.Request) {
	m, err := s.st.GetMeme(r.PathValue("id"), "")
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "meme not found")
		return
	}
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not load meme", err)
		return
	}
	if m.StoredImage != "" {
		http.Redirect(w, r, m.StoredImage, http.StatusTemporaryRedirect)
		return
	}
	if m.TemplateID == "" {
		writeErr(w, http.StatusNotFound, "meme has no template")
		return
	}
	template, err := s.st.GetTemplate(m.TemplateID)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "template not found")
		return
	}
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not load template", err)
		return
	}
	if !strings.HasPrefix(template.ImageURL, "/uploads/") {
		writeErr(w, http.StatusNotFound, "template image not found")
		return
	}
	path := filepath.Join(s.uploads, filepath.FromSlash(strings.TrimPrefix(template.ImageURL, "/uploads/")))

	value, err, _ := s.imageGroup.Do(m.ID, func() (any, error) {
		if cached, ok := s.imageCache.Get(m.ID); ok {
			return cached, nil
		}
		s.imageSem <- struct{}{}
		defer func() { <-s.imageSem }()
		if cached, ok := s.imageCache.Get(m.ID); ok {
			return cached, nil
		}
		var data []byte
		var renderErr error
		if template.Animated {
			data, renderErr = render.RenderGIFFile(path, m.TopText, m.BottomText, m.Style)
			if renderErr != nil {
				// Fall back to a still frame if a GIF is too large or malformed.
				data, renderErr = render.RenderFile(path, m.TopText, m.BottomText, m.Style)
			}
		} else {
			data, renderErr = render.RenderFile(path, m.TopText, m.BottomText, m.Style)
		}
		if renderErr != nil {
			return nil, renderErr
		}
		s.imageCache.Put(m.ID, data)
		return data, nil
	})
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not render meme", err)
		return
	}
	data, ok := value.([]byte)
	if !ok {
		s.fail(w, http.StatusInternalServerError, "could not render meme", errors.New("unexpected cache value"))
		return
	}

	contentType := "image/png"
	extension := "png"
	if template.Animated {
		contentType = "image/gif"
		extension = "gif"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="memebank-%s.%s"`, m.ID, extension))
	_, _ = w.Write(data)
}

func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
	voter := voterID(r)
	if voter == "" {
		writeErr(w, http.StatusBadRequest, "missing or invalid X-Voter-ID header")
		return
	}
	var body struct {
		Value int `json:"value"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	up, down, mine, err := s.st.Vote(r.PathValue("id"), voter, body.Value)
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "meme not found")
		return
	}
	if errors.Is(err, store.ErrInvalidVote) {
		writeErr(w, http.StatusBadRequest, "value must be 1 or -1")
		return
	}
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not record vote", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{
		"upvotes":   up,
		"downvotes": down,
		"score":     up - down,
		"myVote":    mine,
	})
}

func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r.URL.Query().Get("limit"), 8)
	memes, _, err := s.st.ListMemes(store.ListOptions{
		Sort:    store.SortTop,
		Limit:   limit,
		VoterID: voterID(r),
	})
	if err != nil {
		s.fail(w, http.StatusInternalServerError, "could not load leaderboard", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"memes": memes})
}

func (s *Server) fail(w http.ResponseWriter, code int, msg string, err error) {
	s.log.Error(msg, "error", err)
	writeErr(w, code, msg)
}

// --- helpers ---------------------------------------------------------------

func parseMultipart(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return errors.New("upload is too large or malformed (max 12MB)")
	}
	return nil
}

func readImage(src io.Reader) ([]byte, image.Config, string, error) {
	data, err := io.ReadAll(io.LimitReader(src, maxUploadBytes+1))
	if err != nil {
		return nil, image.Config{}, "", errors.New("could not read image")
	}
	if len(data) > maxUploadBytes {
		return nil, image.Config{}, "", errors.New("image is larger than 12MB")
	}
	if len(data) < 16 {
		return nil, image.Config{}, "", errors.New("image file is empty")
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, image.Config{}, "", errors.New("unsupported image format (use png, jpeg, gif or webp)")
	}
	switch format {
	case "png", "jpeg", "gif", "webp":
	default:
		return nil, image.Config{}, "", errors.New("unsupported image format (use png, jpeg, gif or webp)")
	}
	return data, cfg, format, nil
}

func saveUpload(dir string, data []byte, ext string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := randHex(12) + ext
	tmp, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return "", err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		tmp.Close()
		os.Remove(tmpName)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	dst := filepath.Join(dir, name)
	if err := os.Rename(tmpName, dst); err != nil {
		os.Remove(tmpName)
		return "", err
	}
	return name, nil
}

func extensionFor(format string) string {
	if format == "jpeg" {
		return ".jpg"
	}
	return "." + format
}

func randHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func voterID(r *http.Request) string {
	id := strings.TrimSpace(r.Header.Get("X-Voter-ID"))
	if voterIDPattern.MatchString(id) {
		return id
	}
	return ""
}

func intParam(raw string, fallback int) int {
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return n
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("write response failed", "error", err)
	}
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// --- middleware ------------------------------------------------------------

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.log.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration", time.Since(start).Round(time.Microsecond).String(),
		)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Content-Type, X-Voter-ID")
		h.Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.log.Error("panic", "value", rec, "path", r.URL.Path)
				writeErr(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func cacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}

func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := path.Clean(r.URL.Path)
		if clean != "/" {
			file := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(clean, "/")))
			if info, err := os.Stat(file); err == nil && !info.IsDir() {
				if strings.HasPrefix(clean, "/assets/") {
					w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				}
				fs.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}
