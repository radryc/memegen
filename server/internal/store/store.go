// Package store persists templates, memes and votes in SQLite.
package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"path"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/radryc/memebank/server/internal/imghash"
)

// ErrNotFound is returned when a requested row does not exist.
var ErrNotFound = errors.New("not found")

// ErrInvalidVote is returned for vote values other than -1 and 1.
var ErrInvalidVote = errors.New("invalid vote value")

// ErrTemplateInUse is returned when deleting a template that still has memes.
var ErrTemplateInUse = errors.New("template is in use")

// Template is a reusable meme background.
type Template struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	ImageURL       string    `json:"imageUrl"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	Source         string    `json:"source"`
	Animated       bool      `json:"animated"`
	ContentHash    string    `json:"-"`
	PerceptualHash string    `json:"-"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Meme is a published meme.
type Meme struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Author       string    `json:"author"`
	ImageURL     string    `json:"imageUrl"`
	TemplateID   string    `json:"templateId,omitempty"`
	TemplateName string    `json:"templateName,omitempty"`
	TopText      string    `json:"topText"`
	BottomText   string    `json:"bottomText"`
	Style        string    `json:"style"`
	Width        int       `json:"width,omitempty"`
	Height       int       `json:"height,omitempty"`
	Score        int       `json:"score"`
	Upvotes      int       `json:"upvotes"`
	Downvotes    int       `json:"downvotes"`
	MyVote       int       `json:"myVote"`
	Animated     bool      `json:"animated"`
	StoredImage  string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Stats holds board-wide counters.
type Stats struct {
	Memes     int `json:"memes"`
	Votes     int `json:"votes"`
	Templates int `json:"templates"`
}

// Sort modes accepted by ListMemes.
const (
	SortHot = "hot"
	SortNew = "new"
	SortTop = "top"
)

// ListOptions controls meme queries.
type ListOptions struct {
	Sort     string
	Limit    int
	Offset   int
	Search   string
	Template string
	VoterID  string
}

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

// Open opens (and migrates) the database at path.
func Open(path string) (*Store, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// A single connection keeps writes serialized and avoids SQLITE_BUSY for
	// this small workload.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS templates (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	image_url  TEXT NOT NULL,
	width      INTEGER NOT NULL DEFAULT 0,
	height     INTEGER NOT NULL DEFAULT 0,
	source          TEXT NOT NULL DEFAULT 'imgflip',
	content_hash    TEXT NOT NULL DEFAULT '',
	perceptual_hash TEXT NOT NULL DEFAULT '',
	created_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS memes (
	id          TEXT PRIMARY KEY,
	title       TEXT NOT NULL DEFAULT '',
	author      TEXT NOT NULL DEFAULT '',
	image_url   TEXT NOT NULL,
	template_id TEXT REFERENCES templates(id) ON DELETE SET NULL,
	top_text    TEXT NOT NULL DEFAULT '',
	bottom_text TEXT NOT NULL DEFAULT '',
	style       TEXT NOT NULL DEFAULT '{}',
	upvotes     INTEGER NOT NULL DEFAULT 0,
	downvotes   INTEGER NOT NULL DEFAULT 0,
	created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_memes_created ON memes(created_at DESC);

CREATE TABLE IF NOT EXISTS votes (
	meme_id    TEXT NOT NULL REFERENCES memes(id) ON DELETE CASCADE,
	voter_id   TEXT NOT NULL,
	value      INTEGER NOT NULL,
	created_at TEXT NOT NULL,
	PRIMARY KEY (meme_id, voter_id)
);

CREATE TABLE IF NOT EXISTS deleted_templates (
	id         TEXT PRIMARY KEY,
	deleted_at TEXT NOT NULL
);
`)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	// Databases created before upload dedupe existed need the new column.
	if _, err := s.db.Exec(`ALTER TABLE templates ADD COLUMN content_hash TEXT NOT NULL DEFAULT ''`); err != nil &&
		!strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return fmt.Errorf("migrate content hash: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE templates ADD COLUMN perceptual_hash TEXT NOT NULL DEFAULT ''`); err != nil &&
		!strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return fmt.Errorf("migrate perceptual hash: %w", err)
	}
	if _, err := s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_templates_hash ON templates(content_hash)`); err != nil {
		return fmt.Errorf("migrate content hash index: %w", err)
	}
	return nil
}

// NewID returns a random 16 character hex identifier.
func NewID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// IsAnimatedURL reports whether a stored image is a GIF.
func IsAnimatedURL(url string) bool {
	return strings.EqualFold(path.Ext(url), ".gif")
}

// CountTemplates reports how many templates exist.
func (s *Store) CountTemplates() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM templates`).Scan(&n)
	return n, err
}

// IsTemplateDeleted reports whether a template id was removed by an admin.
// Seeders use this so deleted templates are not recreated on restart.
func (s *Store) IsTemplateDeleted(id string) (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM deleted_templates WHERE id = ?`, id).Scan(&n)
	return n > 0, err
}

// CountAnimatedTemplates reports how many GIF templates exist.
func (s *Store) CountAnimatedTemplates() (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM templates WHERE lower(image_url) LIKE '%.gif'`).Scan(&n)
	return n, err
}

// ListTemplates returns templates, popular ones first. media may be
// "gif", "image" or empty for both.
func (s *Store) ListTemplates(source, search, media string) ([]Template, error) {
	q := `SELECT id, name, image_url, width, height, source, created_at FROM templates`
	var where []string
	var args []any
	if source != "" && source != "all" {
		where = append(where, "source = ?")
		args = append(args, source)
	}
	switch media {
	case "gif":
		where = append(where, "lower(image_url) LIKE '%.gif'")
	case "image":
		where = append(where, "lower(image_url) NOT LIKE '%.gif'")
	}
	if search != "" {
		where = append(where, "(name LIKE ? OR source LIKE ?)")
		like := "%" + search + "%"
		args = append(args, like, like)
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	q += " ORDER BY CASE source WHEN 'imgflip' THEN 0 ELSE 1 END, created_at ASC, rowid ASC"

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Template{}
	for rows.Next() {
		var t Template
		var created string
		if err := rows.Scan(&t.ID, &t.Name, &t.ImageURL, &t.Width, &t.Height, &t.Source, &created); err != nil {
			return nil, err
		}
		t.CreatedAt = parseTime(created)
		t.Animated = IsAnimatedURL(t.ImageURL)
		out = append(out, t)
	}
	return out, rows.Err()
}

// GetTemplate returns a single template or ErrNotFound.
func (s *Store) GetTemplate(id string) (Template, error) {
	var t Template
	var created string
	err := s.db.QueryRow(
		`SELECT id, name, image_url, width, height, source, created_at FROM templates WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &t.ImageURL, &t.Width, &t.Height, &t.Source, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	if err != nil {
		return Template{}, err
	}
	t.CreatedAt = parseTime(created)
	t.Animated = IsAnimatedURL(t.ImageURL)
	return t, nil
}

// CreateTemplate inserts a template.
func (s *Store) CreateTemplate(t Template) error {
	if t.ID == "" {
		t.ID = NewID()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(
		`INSERT INTO templates (id, name, image_url, width, height, source, content_hash, perceptual_hash, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.ImageURL, t.Width, t.Height, t.Source, t.ContentHash, t.PerceptualHash,
		t.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	return err
}

// FindSimilarTemplate returns the closest visually similar template within the
// dHash threshold. Animated and static templates are kept separate so a still
// frame of a GIF does not silently replace an animated template.
func (s *Store) FindSimilarTemplate(hash string, aspect float64, animated bool) (Template, error) {
	if hash == "" {
		return Template{}, ErrNotFound
	}
	pattern := "NOT LIKE"
	if animated {
		pattern = "LIKE"
	}
	rows, err := s.db.Query(
		`SELECT id, name, image_url, width, height, source, content_hash, perceptual_hash, created_at
		 FROM templates WHERE perceptual_hash <> '' AND lower(image_url) ` + pattern + ` '%.gif'`,
	)
	if err != nil {
		return Template{}, err
	}
	defer rows.Close()

	bestDistance := imghash.MaxSimilarDistance + 1
	var best Template
	for rows.Next() {
		var t Template
		var created string
		if err := rows.Scan(&t.ID, &t.Name, &t.ImageURL, &t.Width, &t.Height, &t.Source,
			&t.ContentHash, &t.PerceptualHash, &created); err != nil {
			return Template{}, err
		}
		if aspect > 0 && t.Width > 0 && t.Height > 0 {
			ta := float64(t.Width) / float64(t.Height)
			if math.Abs(ta-aspect) > 0.035 {
				continue
			}
		}
		dist := imghash.Distance(hash, t.PerceptualHash)
		if dist < bestDistance {
			bestDistance = dist
			t.CreatedAt = parseTime(created)
			best = t
		}
	}
	if err := rows.Err(); err != nil {
		return Template{}, err
	}
	if best.ID == "" {
		return Template{}, ErrNotFound
	}
	best.Animated = IsAnimatedURL(best.ImageURL)
	return best, nil
}

// MemeCountsByTemplate returns how many memes use each template.
func (s *Store) MemeCountsByTemplate() (map[string]int, error) {
	rows, err := s.db.Query(
		`SELECT template_id, COUNT(*) FROM memes WHERE template_id IS NOT NULL GROUP BY template_id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := make(map[string]int)
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		counts[id] = n
	}
	return counts, rows.Err()
}

// CountMemesByTemplate reports how many memes reference a template.
func (s *Store) CountMemesByTemplate(id string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM memes WHERE template_id = ?`, id).Scan(&n)
	return n, err
}

// DeleteTemplate removes a template. If memes reference it, they must be
// deleted too by passing withMemes. A tombstone prevents seeders from adding
// the template back on the next restart.
func (s *Store) DeleteTemplate(id string, withMemes bool) ([]Meme, Template, error) {
	var deletedMemes []Meme
	var template Template

	tx, err := s.db.Begin()
	if err != nil {
		return nil, template, err
	}
	defer tx.Rollback()

	var created string
	err = tx.QueryRow(
		`SELECT id, name, image_url, width, height, source, created_at FROM templates WHERE id = ?`, id,
	).Scan(&template.ID, &template.Name, &template.ImageURL, &template.Width, &template.Height,
		&template.Source, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, template, ErrNotFound
	}
	if err != nil {
		return nil, template, err
	}
	template.CreatedAt = parseTime(created)
	template.Animated = IsAnimatedURL(template.ImageURL)

	rows, err := tx.Query(`SELECT id, title, author, image_url FROM memes WHERE template_id = ?`, id)
	if err != nil {
		return nil, template, err
	}
	for rows.Next() {
		var m Meme
		var stored string
		if err := rows.Scan(&m.ID, &m.Title, &m.Author, &stored); err != nil {
			rows.Close()
			return nil, template, err
		}
		m.StoredImage = stored
		m.ImageURL = stored
		deletedMemes = append(deletedMemes, m)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, template, err
	}
	rows.Close()

	if len(deletedMemes) > 0 && !withMemes {
		return deletedMemes, template, ErrTemplateInUse
	}
	if len(deletedMemes) > 0 {
		if _, err := tx.Exec(`DELETE FROM memes WHERE template_id = ?`, id); err != nil {
			return nil, template, err
		}
	}
	if _, err := tx.Exec(`DELETE FROM templates WHERE id = ?`, id); err != nil {
		return nil, template, err
	}
	if _, err := tx.Exec(
		`INSERT OR REPLACE INTO deleted_templates (id, deleted_at) VALUES (?, ?)`,
		id, time.Now().UTC().Format(time.RFC3339Nano),
	); err != nil {
		return nil, template, err
	}
	if err := tx.Commit(); err != nil {
		return nil, template, err
	}
	return deletedMemes, template, nil
}

// ListStoredMemes returns memes that still have a rendered file on disk. Used
// by the admin migration to move legacy memes to on-demand rendering.
func (s *Store) ListStoredMemes() ([]Meme, error) {
	rows, err := s.db.Query(`
SELECT m.id, m.title, m.author, m.image_url, COALESCE(m.template_id, ''), COALESCE(t.name, ''),
       COALESCE(t.image_url, ''), COALESCE(t.width, 0), COALESCE(t.height, 0),
       m.top_text, m.bottom_text, m.style, m.upvotes, m.downvotes, m.created_at, 0
FROM memes m
LEFT JOIN templates t ON t.id = m.template_id
WHERE m.image_url <> '' AND m.template_id IS NOT NULL
ORDER BY m.created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Meme
	for rows.Next() {
		m, err := scanMeme(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ClearMemeImage marks a meme as spec-rendered.
func (s *Store) ClearMemeImage(id string) error {
	_, err := s.db.Exec(`UPDATE memes SET image_url = '' WHERE id = ?`, id)
	return err
}

// CountTemplatesByImageURL reports how many templates reference an image file.
func (s *Store) CountTemplatesByImageURL(url string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM templates WHERE image_url = ?`, url).Scan(&n)
	return n, err
}

// CountMemesByImageURL reports how many memes still reference a stored file.
func (s *Store) CountMemesByImageURL(url string) (int, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM memes WHERE image_url = ?`, url).Scan(&n)
	return n, err
}

// DeleteMeme removes a meme (votes cascade) and returns its stored image URL
// so the caller can delete the file.
func (s *Store) DeleteMeme(id string) (Meme, error) {
	m, err := s.GetMeme(id, "")
	if err != nil {
		return Meme{}, err
	}
	res, err := s.db.Exec(`DELETE FROM memes WHERE id = ?`, id)
	if err != nil {
		return Meme{}, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Meme{}, err
	}
	if n == 0 {
		return Meme{}, ErrNotFound
	}
	return m, nil
}

// ListTemplatesMissingFingerprints returns templates that still need exact or
// perceptual fingerprints computed.
func (s *Store) ListTemplatesMissingFingerprints() ([]Template, error) {
	rows, err := s.db.Query(
		`SELECT id, image_url, content_hash, perceptual_hash FROM templates
		 WHERE content_hash = '' OR perceptual_hash = ''`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Template{}
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.ImageURL, &t.ContentHash, &t.PerceptualHash); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SetTemplateFingerprints stores exact and perceptual hashes for a template.
func (s *Store) SetTemplateFingerprints(id, exact, perceptual string) error {
	_, err := s.db.Exec(
		`UPDATE templates SET content_hash = ?, perceptual_hash = ? WHERE id = ?`,
		exact, perceptual, id,
	)
	return err
}

// GetTemplateByHash finds a template with an identical image fingerprint.
func (s *Store) GetTemplateByHash(hash string) (Template, error) {
	if hash == "" {
		return Template{}, ErrNotFound
	}
	var t Template
	var created string
	err := s.db.QueryRow(
		`SELECT id, name, image_url, width, height, source, content_hash, created_at
		 FROM templates WHERE content_hash = ? ORDER BY created_at ASC LIMIT 1`, hash,
	).Scan(&t.ID, &t.Name, &t.ImageURL, &t.Width, &t.Height, &t.Source, &t.ContentHash, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Template{}, ErrNotFound
	}
	if err != nil {
		return Template{}, err
	}
	t.CreatedAt = parseTime(created)
	t.Animated = IsAnimatedURL(t.ImageURL)
	return t, nil
}

// ListTemplatesMissingHash returns templates whose fingerprint still needs to
// be computed (existing databases from before dedupe support).
func (s *Store) ListTemplatesMissingHash() ([]Template, error) {
	rows, err := s.db.Query(`SELECT id, image_url FROM templates WHERE content_hash = ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Template{}
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.ImageURL); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SetTemplateHash stores an image fingerprint for a template.
func (s *Store) SetTemplateHash(id, hash string) error {
	_, err := s.db.Exec(`UPDATE templates SET content_hash = ? WHERE id = ?`, hash, id)
	return err
}

// GetMeme returns a meme with the current vote for voterID (0 if none).
func (s *Store) GetMeme(id, voterID string) (Meme, error) {
	row := s.db.QueryRow(`
SELECT m.id, m.title, m.author, m.image_url, COALESCE(m.template_id, ''), COALESCE(t.name, ''),
       COALESCE(t.image_url, ''), COALESCE(t.width, 0), COALESCE(t.height, 0), m.top_text, m.bottom_text, m.style, m.upvotes, m.downvotes, m.created_at,
       COALESCE((SELECT value FROM votes WHERE meme_id = m.id AND voter_id = ?), 0)
FROM memes m
LEFT JOIN templates t ON t.id = m.template_id
WHERE m.id = ?`, voterID, id)

	m, err := scanMeme(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Meme{}, ErrNotFound
	}
	if err != nil {
		return Meme{}, err
	}
	return m, nil
}

// ListMemes returns memes matching opts plus whether more rows exist.
func (s *Store) ListMemes(opts ListOptions) ([]Meme, bool, error) {
	if opts.Limit <= 0 || opts.Limit > 100 {
		opts.Limit = 24
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}
	if opts.Sort == "" {
		opts.Sort = SortHot
	}

	// Hot ranking needs a computed score, so fetch a wider recent window and
	// sort it in Go. Other modes are ordered fully by SQLite.
	limit := opts.Limit + 1
	if opts.Sort == SortHot {
		limit = 1000
		opts.Offset = 0
	}

	q := `
SELECT m.id, m.title, m.author, m.image_url, COALESCE(m.template_id, ''), COALESCE(t.name, ''),
       COALESCE(t.image_url, ''), COALESCE(t.width, 0), COALESCE(t.height, 0), m.top_text, m.bottom_text, m.style, m.upvotes, m.downvotes, m.created_at,
       COALESCE((SELECT value FROM votes WHERE meme_id = m.id AND voter_id = ?), 0)
FROM memes m
LEFT JOIN templates t ON t.id = m.template_id`
	var where []string
	var args []any
	args = append(args, opts.VoterID)
	if opts.Template != "" {
		where = append(where, "m.template_id = ?")
		args = append(args, opts.Template)
	}
	if opts.Search != "" {
		where = append(where, "(m.title LIKE ? OR m.top_text LIKE ? OR m.bottom_text LIKE ? OR m.author LIKE ? OR COALESCE(t.name, '') LIKE ?)")
		like := "%" + opts.Search + "%"
		args = append(args, like, like, like, like, like)
	}
	if len(where) > 0 {
		q += " WHERE " + strings.Join(where, " AND ")
	}
	switch opts.Sort {
	case SortTop:
		q += " ORDER BY (m.upvotes - m.downvotes) DESC, m.created_at DESC"
	default: // new and hot both start from a recent window
		q += " ORDER BY m.created_at DESC"
	}
	q += " LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	memes := []Meme{}
	for rows.Next() {
		m, err := scanMeme(rows)
		if err != nil {
			return nil, false, err
		}
		memes = append(memes, m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}

	if opts.Sort == SortHot {
		now := time.Now()
		for i := range memes {
			memes[i].Score = memes[i].Upvotes - memes[i].Downvotes
			sortHot(memes, now)
		}
		if opts.Offset >= len(memes) {
			return []Meme{}, false, nil
		}
		end := opts.Offset + opts.Limit
		if end > len(memes) {
			end = len(memes)
		}
		return memes[opts.Offset:end], end < len(memes), nil
	}

	hasMore := len(memes) > opts.Limit
	if hasMore {
		memes = memes[:opts.Limit]
	}
	return memes, hasMore, nil
}

// sortHot orders memes by a Hacker News style engagement/recency score.
func sortHot(memes []Meme, now time.Time) {
	rank := make(map[string]float64, len(memes))
	for _, m := range memes {
		age := now.Sub(m.CreatedAt).Hours()
		if age < 0 {
			age = 0
		}
		rank[m.ID] = (float64(m.Score) + 1) / math.Pow(age+2, 1.35)
	}
	// insertion sort keeps this allocation-free; the window is capped at 1000.
	for i := 1; i < len(memes); i++ {
		for j := i; j > 0 && rank[memes[j].ID] > rank[memes[j-1].ID]; j-- {
			memes[j], memes[j-1] = memes[j-1], memes[j]
		}
	}
}

// CreateMeme inserts a published meme.
func (s *Store) CreateMeme(m Meme) (Meme, error) {
	if m.ID == "" {
		m.ID = NewID()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	var templateID any
	if m.TemplateID != "" {
		templateID = m.TemplateID
	}
	_, err := s.db.Exec(`
INSERT INTO memes (id, title, author, image_url, template_id, top_text, bottom_text, style, upvotes, downvotes, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?)`,
		m.ID, m.Title, m.Author, m.ImageURL, templateID, m.TopText, m.BottomText, m.Style,
		m.CreatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return Meme{}, err
	}
	return s.GetMeme(m.ID, "")
}

// Vote applies value (-1 or 1) for voterID, toggling an identical vote off.
// It returns the updated tallies and the voter's resulting vote.
func (s *Store) Vote(memeID, voterID string, value int) (up, down, mine int, err error) {
	if value != 1 && value != -1 {
		return 0, 0, 0, ErrInvalidVote
	}
	tx, err := s.db.Begin()
	if err != nil {
		return 0, 0, 0, err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM memes WHERE id = ?`, memeID).Scan(&exists); err != nil {
		return 0, 0, 0, err
	}
	if exists == 0 {
		return 0, 0, 0, ErrNotFound
	}

	var current int
	err = tx.QueryRow(`SELECT value FROM votes WHERE meme_id = ? AND voter_id = ?`, memeID, voterID).Scan(&current)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = tx.Exec(`INSERT INTO votes (meme_id, voter_id, value, created_at) VALUES (?, ?, ?, ?)`,
			memeID, voterID, value, time.Now().UTC().Format(time.RFC3339Nano))
		mine = value
	case err != nil:
		return 0, 0, 0, err
	case current == value:
		_, err = tx.Exec(`DELETE FROM votes WHERE meme_id = ? AND voter_id = ?`, memeID, voterID)
		mine = 0
	default:
		_, err = tx.Exec(`UPDATE votes SET value = ?, created_at = ? WHERE meme_id = ? AND voter_id = ?`,
			value, time.Now().UTC().Format(time.RFC3339Nano), memeID, voterID)
		mine = value
	}
	if err != nil {
		return 0, 0, 0, err
	}

	if err := tx.QueryRow(`
SELECT COALESCE(SUM(CASE WHEN value = 1 THEN 1 ELSE 0 END), 0),
       COALESCE(SUM(CASE WHEN value = -1 THEN 1 ELSE 0 END), 0)
FROM votes WHERE meme_id = ?`, memeID).Scan(&up, &down); err != nil {
		return 0, 0, 0, err
	}
	if _, err := tx.Exec(`UPDATE memes SET upvotes = ?, downvotes = ? WHERE id = ?`, up, down, memeID); err != nil {
		return 0, 0, 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, 0, err
	}
	return up, down, mine, nil
}

// Stats returns aggregate counters.
func (s *Store) Stats() (Stats, error) {
	var st Stats
	err := s.db.QueryRow(`
SELECT (SELECT COUNT(*) FROM memes),
       (SELECT COUNT(*) FROM votes),
       (SELECT COUNT(*) FROM templates)`).Scan(&st.Memes, &st.Votes, &st.Templates)
	return st, err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanMeme(sc scanner) (Meme, error) {
	var m Meme
	var created string
	var templateImage string
	err := sc.Scan(&m.ID, &m.Title, &m.Author, &m.ImageURL, &m.TemplateID, &m.TemplateName,
		&templateImage, &m.Width, &m.Height, &m.TopText, &m.BottomText, &m.Style,
		&m.Upvotes, &m.Downvotes, &created, &m.MyVote)
	if err != nil {
		return Meme{}, err
	}
	m.CreatedAt = parseTime(created)
	m.Score = m.Upvotes - m.Downvotes
	m.StoredImage = m.ImageURL
	if m.ImageURL == "" {
		// New memes store only their spec; the image is rendered on demand.
		m.ImageURL = "/api/memes/" + m.ID + "/image"
	}
	m.Animated = IsAnimatedURL(m.ImageURL) || IsAnimatedURL(templateImage)
	return m, nil
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}
