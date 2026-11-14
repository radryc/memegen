package seed

import (
	"image/gif"
	"os"
	"path/filepath"
	"testing"

	"github.com/radryc/memebank/server/internal/store"
)

func TestSeedGeneratedGIFs(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	dir := t.TempDir()
	inserted, err := seedGeneratedGIFs(st, dir)
	if err != nil {
		t.Fatalf("seed gifs: %v", err)
	}
	if inserted != 5 {
		t.Fatalf("inserted = %d", inserted)
	}

	templates, err := st.ListTemplates("", "", "gif")
	if err != nil {
		t.Fatalf("list gifs: %v", err)
	}
	if len(templates) != 5 {
		t.Fatalf("gif templates = %d", len(templates))
	}
	for _, template := range templates {
		if !template.Animated {
			t.Fatalf("template %s is not marked animated", template.ID)
		}
	}

	file, err := os.Open(filepath.Join(dir, "gif-canvas-1.gif"))
	if err != nil {
		t.Fatalf("open gif: %v", err)
	}
	defer file.Close()
	decoded, err := gif.DecodeAll(file)
	if err != nil {
		t.Fatalf("decode gif: %v", err)
	}
	if len(decoded.Image) != gifFrames {
		t.Fatalf("frames = %d", len(decoded.Image))
	}
	if decoded.Config.Width != gifWidth || decoded.Config.Height != gifHeight {
		t.Fatalf("size = %dx%d", decoded.Config.Width, decoded.Config.Height)
	}
}
