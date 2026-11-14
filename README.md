# memebank

Make it. Post it. Get roasted.

A meme board and generator: pick a template or upload your own image/GIF, add captions in the live editor, publish to the board, and let the crowd score it. Think Imgflip meets a community meme leaderboard.

Built with **Go** (net/http + SQLite) and **React/TypeScript** (Vite + Tailwind CSS).

## Features

- **Template library** seeded from the Imgflip public API (72 classics, with Change My Mind aliased as Convince Me for search), plus handpicked additions (Facepalm, Young Thug at Computer, First World Problems, Elon Musk…) and generated blank canvases.
- **Animated GIF templates** — generated animated canvases out of the box, GIF uploads, animated previews, and real animated GIF memes.
- **Meme editor** with live canvas preview, top/bottom captions, font scaling, text colours, outline/uppercase toggles, random caption generator, PNG/GIF export.
- **Upload or paste (⌘V / Ctrl+V) images and GIFs** and immediately use them as templates; uploads join the community library.
- **Duplicate-proof uploads** — templates are fingerprinted with SHA-256 and a 256-bit perceptual dHash. The same file, a resized copy, or a re-encoded version all reuse the existing template instead of cluttering the library.
- **Scoring** with optimistic up/down votes, one vote per browser per meme, vote toggling, hot/new/top sorting, and a full leaderboard.
- **Site-wide search** across memes (titles, captions, creators, template names) and templates, with `/` as a keyboard shortcut.
- **Local admin CLI** — list/delete templates and memes, cascade votes and files, and migrate legacy renders to specs.
- **No accounts** — anonymous browser voting id stored in localStorage.
- **Meme detail pages** with score breakdown, download, copy link, remix, and related memes.
- Responsive dark UI with a mobile bottom nav.

## Quick start (development)

Requirements: Go 1.26+ and Node 20.19+ (or 22.12+).

```bash
# terminal 1 — API on :8080, seeds templates on first run
make dev-api

# terminal 2 — Vite dev server on :5173 (proxies /api and /uploads)
make dev-web
```

Open http://localhost:5173.

The first backend start downloads ~72 template images from Imgflip and generates five animated GIF canvases. This takes a few seconds and only happens once per data directory.

## Production build

```bash
make build     # builds web/dist and bin/memebank
make run       # serves API + UI on http://localhost:8080
```

Or manually:

```bash
cd web && npm ci && npm run build
cd server && go build -o ../bin/memebank .
./bin/memebank -addr :8080 -data ./data -static ./web/dist
```

### Configuration

| Flag | Env | Default | Purpose |
| --- | --- | --- | --- |
| `-addr` | `MEMEBANK_ADDR` | `:8080` | Listen address |
| `-data` | `MEMEBANK_DATA` | `data` | SQLite DB and uploads |
| `-static` | `MEMEBANK_STATIC` | auto-detect | Built frontend directory |
| `-no-seed` | — | `false` | Skip template seeding |
| `-debug` | `MEMEBANK_DEBUG=1` | `false` | Debug logs |

Optional: set `GIPHY_API_KEY` to seed ~10 trending GIF templates from GIPHY instead of generated animated canvases. GIF templates pulled this way are subject to GIPHY's terms and attribution requirements. Without a key, everything still works with generated GIFs and uploads.

## How scoring works

- Each browser gets a random voter id in `localStorage`.
- Upvote/downvote is toggled: clicking the same arrow again removes the vote.
- Net score = upvotes − downvotes.
- `Hot` combines score with recency (Hacker News style decay), `New` is chronological, `Top` is all-time score.

## API

All endpoints are JSON unless noted. Votes require an `X-Voter-ID` header (8–64 chars, `A-Za-z0-9_-`).

| Method | Path | Description |
| --- | --- | --- |
| GET | `/api/health` | Health check |
| GET | `/api/stats` | Meme/vote/template counts |
| GET | `/api/templates?source=&media=&q=` | Templates; `source` is `imgflip`, `upload`, `generated`, `giphy`; `media` is `gif` or `image` |
| POST | `/api/templates` | Multipart: `image`, `name`. Returns `{template, duplicate}`; identical bytes return the existing template with `duplicate: true` |
| GET | `/api/memes?sort=hot\|new\|top&q=&template=&limit=&offset=` | Meme feed |
| POST | `/api/memes` | Multipart: `title`, `author`, `topText`, `bottomText`, `templateId`, `style` (JSON). `image` is optional and only needed for legacy/clients that pre-render |
| GET | `/api/memes/{id}` | Single meme |
| GET | `/api/memes/{id}/image` | Render the meme on demand (PNG or animated GIF) |
| POST | `/api/memes/{id}/vote` | JSON `{"value": 1}` or `{"value": -1}` |
| GET | `/api/leaderboard?limit=8` | Top-scored memes |

Uploads are served from `/uploads/...` with immutable caching. Duplicate detection uses SHA-256 for exact matches and an aspect-aware 256-bit dHash for visually identical images (including resized/re-encoded copies, within a Hamming distance of 14/256). Animated and static templates are compared separately.

### Storage model

Memes are stored as a **spec**, not as image files:

- `template_id` + `top_text` + `bottom_text` + `style` JSON
- The image is rendered on demand at `/api/memes/{id}/image`
- Static templates render to PNG with the embedded Anton font
- GIF templates render to an animated GIF with per-frame disposal handling and median-cut quantization
- Rendered images are cached in memory (64MB), so durable storage only contains templates and uploads
- Existing memes created before this change keep their stored files until you run `memebank-admin memes migrate-spec -yes` (or until they are migrated manually); new memes never write to `uploads/memes`

The editor still uses the canvas for the live preview and for the optional "download draft" button. Publishing is a tiny multipart request with no image upload.

Animated GIFs are CPU-intensive on the first request (roughly 1–2s for a 24-frame template), then served from cache. Static renders take well under a second. Limits: 150 frames and 900px on the longest side. If a GIF cannot be rendered, the endpoint falls back to a still frame.

## Project layout

```
server/
  main.go                 flags, shutdown, static serving
  internal/api/           HTTP handlers, uploads, SPA
  internal/store/         SQLite schema + queries
  internal/seed/          Imgflip seeding, generated GIFs, Giphy
web/
  src/components/         layout, cards, votes, icons
  src/pages/              feed, editor, templates, search, detail, leaderboard
  src/lib/                API client, canvas renderer, GIF compositor
  src/index.css           Tailwind theme + design system
```

## Admin CLI

Administration is local-only: there is no admin HTTP endpoint. The CLI talks
directly to the SQLite database and uploads directory.

```bash
make build-admin

# Development data (server/data):
make admin ARGS="templates list"

# Production data (./data):
bin/memebank-admin -data ./data templates list
```

Commands:

```bash
memebank-admin [-data DIR] templates list [-q TEXT]
memebank-admin [-data DIR] templates delete ID [-with-memes] [-yes]
memebank-admin [-data DIR] memes list [-q TEXT] [-template ID] [-limit N]
memebank-admin [-data DIR] memes delete ID [-yes]
memebank-admin [-data DIR] memes migrate-spec [-yes]
memebank-admin [-data DIR] stats
```

Safety and behavior:

- Without `-yes` every destructive command is a dry run.
- Deleting a template that still has memes is refused unless `-with-memes` is
  passed; that deletes the memes (and their votes) too.
- Template, meme, and rendered-image files are removed from disk when no
  longer referenced.
- Deleted template IDs are tombstoned, so the seeder will not resurrect them
  on the next restart.
- `memes migrate-spec` converts legacy stored renders into spec-only memes
  (template + captions + style) and deletes their image files. Memes without a
  template or captions are left untouched.
- The CLI is safe to run against a live server thanks to SQLite WAL, though
  browsers may keep already-cached images until their cache expires.

Example: remove an unused template and convert old stored memes:

```bash
bin/memebank-admin -data ./data templates delete curated-elon-dancing -yes
bin/memebank-admin -data ./data memes migrate-spec -yes
```

## Development commands

```bash
make test      # Go tests + TypeScript typecheck
make fmt       # gofmt
make vet       # go vet
```

## Attribution

Classic meme templates come from the public [Imgflip API](https://api.imgflip.com/). Template images remain the property of their respective owners. Optional GIPHY seeding requires your own API key and compliance with GIPHY's terms.

## License

Apache License 2.0. See [LICENSE](LICENSE).
