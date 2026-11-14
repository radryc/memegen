package seed

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/radryc/memebank/server/internal/store"
)

func TestBackfillTemplateHashes(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	data := []byte("pretend this is a png")
	uploads := t.TempDir()
	if err := os.MkdirAll(filepath.Join(uploads, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(uploads, "templates", "x.png"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateTemplate(store.Template{
		ID:        "x",
		Name:      "X",
		ImageURL:  "/uploads/templates/x.png",
		Width:     100,
		Height:    100,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	backfillTemplateFingerprints(st, uploads, log)

	sum := sha256.Sum256(data)
	want := hex.EncodeToString(sum[:])
	found, err := st.GetTemplateByHash(want)
	if err != nil || found.ID != "x" {
		t.Fatalf("hashed template = %+v err=%v", found, err)
	}
}
