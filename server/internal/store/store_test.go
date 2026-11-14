package store

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func mustTemplate(t *testing.T, st *Store, id, url string) {
	t.Helper()
	if err := st.CreateTemplate(Template{
		ID:        id,
		Name:      "Template " + id,
		ImageURL:  url,
		Width:     800,
		Height:    600,
		Source:    "test",
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create template: %v", err)
	}
}

func TestVoteToggleAndTally(t *testing.T) {
	st := testStore(t)
	mustTemplate(t, st, "t1", "/uploads/templates/t1.png")
	meme, err := st.CreateMeme(Meme{Title: "hi", ImageURL: "/uploads/memes/m1.png", TemplateID: "t1"})
	if err != nil {
		t.Fatalf("create meme: %v", err)
	}

	up, down, mine, err := st.Vote(meme.ID, "voter-one", 1)
	if err != nil || up != 1 || down != 0 || mine != 1 {
		t.Fatalf("first vote = %d/%d/%d err=%v", up, down, mine, err)
	}

	_, _, mine, err = st.Vote(meme.ID, "voter-one", 1)
	if err != nil || mine != 0 {
		t.Fatalf("toggle off = %d err=%v", mine, err)
	}

	up, down, mine, err = st.Vote(meme.ID, "voter-one", -1)
	if err != nil || up != 0 || down != 1 || mine != -1 {
		t.Fatalf("downvote = %d/%d/%d err=%v", up, down, mine, err)
	}

	up, down, _, err = st.Vote(meme.ID, "voter-two", 1)
	if err != nil || up != 1 || down != 1 {
		t.Fatalf("second voter = %d/%d err=%v", up, down, err)
	}

	if _, _, _, err := st.Vote(meme.ID, "voter-two", 0); !errors.Is(err, ErrInvalidVote) {
		t.Fatalf("invalid vote err = %v", err)
	}
	if _, _, _, err := st.Vote("nope", "voter-two", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing meme err = %v", err)
	}

	got, err := st.GetMeme(meme.ID, "voter-one")
	if err != nil {
		t.Fatalf("get meme: %v", err)
	}
	if got.Score != 0 || got.MyVote != -1 {
		t.Fatalf("get meme score=%d myVote=%d", got.Score, got.MyVote)
	}
}

func TestTemplateMediaFilter(t *testing.T) {
	st := testStore(t)
	mustTemplate(t, st, "png", "/uploads/templates/a.png")
	mustTemplate(t, st, "gif", "/uploads/templates/b.gif")

	gifs, err := st.ListTemplates("", "", "gif")
	if err != nil {
		t.Fatalf("list gifs: %v", err)
	}
	if len(gifs) != 1 || gifs[0].ID != "gif" || !gifs[0].Animated {
		t.Fatalf("gif filter = %+v", gifs)
	}

	images, err := st.ListTemplates("", "", "image")
	if err != nil {
		t.Fatalf("list images: %v", err)
	}
	if len(images) != 1 || images[0].ID != "png" || images[0].Animated {
		t.Fatalf("image filter = %+v", images)
	}

	if !IsAnimatedURL("https://x/y.GIF") || IsAnimatedURL("https://x/y.png") {
		t.Fatal("IsAnimatedURL mismatch")
	}
}

func TestListMemesSortAndSearch(t *testing.T) {
	st := testStore(t)
	mustTemplate(t, st, "drake", "/uploads/templates/drake.png")
	now := time.Now().UTC()

	old, err := st.CreateMeme(Meme{
		Title:      "Alpha code",
		ImageURL:   "/uploads/memes/old.png",
		TemplateID: "drake",
		CreatedAt:  now.Add(-2 * time.Hour),
	})
	if err != nil {
		t.Fatalf("create old: %v", err)
	}
	fresh, err := st.CreateMeme(Meme{
		Title:      "Beta cat",
		ImageURL:   "/uploads/memes/new.gif",
		TemplateID: "drake",
		CreatedAt:  now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("create fresh: %v", err)
	}

	if _, _, _, err := st.Vote(old.ID, "a", 1); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := st.Vote(old.ID, "b", 1); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := st.Vote(fresh.ID, "a", -1); err != nil {
		t.Fatal(err)
	}

	top, _, err := st.ListMemes(ListOptions{Sort: SortTop, Limit: 10})
	if err != nil {
		t.Fatalf("top: %v", err)
	}
	if len(top) != 2 || top[0].ID != old.ID {
		t.Fatalf("top order = %+v", top)
	}

	freshList, _, err := st.ListMemes(ListOptions{Sort: SortNew, Limit: 10})
	if err != nil || freshList[0].ID != fresh.ID {
		t.Fatalf("new order = %+v err=%v", freshList, err)
	}
	if !freshList[0].Animated {
		t.Fatal("gif meme should be animated")
	}

	results, _, err := st.ListMemes(ListOptions{Sort: SortHot, Limit: 10, Search: "drake"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("search by template name = %d", len(results))
	}
}

func TestTemplateHashDedup(t *testing.T) {
	st := testStore(t)
	hash := "f0e1d2c3b4a5968778695a4b3c2d1e0f"
	if err := st.CreateTemplate(Template{
		ID:          "hashed",
		Name:        "Hashed",
		ImageURL:    "/uploads/templates/hashed.png",
		Width:       800,
		Height:      600,
		ContentHash: hash,
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create hashed template: %v", err)
	}
	if err := st.CreateTemplate(Template{
		ID:        "legacy",
		Name:      "Legacy",
		ImageURL:  "/uploads/templates/legacy.png",
		Width:     800,
		Height:    600,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("create legacy template: %v", err)
	}

	found, err := st.GetTemplateByHash(hash)
	if err != nil || found.ID != "hashed" {
		t.Fatalf("lookup by hash = %+v err=%v", found, err)
	}
	if _, err := st.GetTemplateByHash("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing hash err = %v", err)
	}

	missing, err := st.ListTemplatesMissingHash()
	if err != nil || len(missing) != 1 || missing[0].ID != "legacy" {
		t.Fatalf("missing hashes = %+v err=%v", missing, err)
	}
	if err := st.SetTemplateHash("legacy", "abc123"); err != nil {
		t.Fatalf("set hash: %v", err)
	}
	missing, err = st.ListTemplatesMissingHash()
	if err != nil || len(missing) != 0 {
		t.Fatalf("missing hashes after backfill = %+v err=%v", missing, err)
	}
}

func TestDeleteMemeAndTemplate(t *testing.T) {
	st := testStore(t)
	mustTemplate(t, st, "t1", "/uploads/templates/t1.png")
	mustTemplate(t, st, "t2", "/uploads/templates/t2.png")

	m1, err := st.CreateMeme(Meme{Title: "one", ImageURL: "/uploads/memes/one.png", TemplateID: "t1"})
	if err != nil {
		t.Fatal(err)
	}
	m2, err := st.CreateMeme(Meme{Title: "two", ImageURL: "/uploads/memes/two.png", TemplateID: "t1"})
	if err != nil {
		t.Fatal(err)
	}
	m3, err := st.CreateMeme(Meme{Title: "three", TemplateID: "t2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := st.Vote(m1.ID, "voter", 1); err != nil {
		t.Fatal(err)
	}

	deleted, err := st.DeleteMeme(m1.ID)
	if err != nil {
		t.Fatalf("delete meme: %v", err)
	}
	if deleted.StoredImage != "/uploads/memes/one.png" || deleted.Title != "one" {
		t.Fatalf("deleted meme = %+v", deleted)
	}
	if _, err := st.GetMeme(m1.ID, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted meme still exists: %v", err)
	}
	stats, err := st.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Votes != 0 {
		t.Fatalf("votes after delete = %d", stats.Votes)
	}

	if _, _, err := st.DeleteTemplate("t2", false); !errors.Is(err, ErrTemplateInUse) {
		t.Fatalf("template in use err = %v", err)
	}
	memes, template, err := st.DeleteTemplate("t2", true)
	if err != nil {
		t.Fatalf("delete template: %v", err)
	}
	if template.ID != "t2" || len(memes) != 1 || memes[0].ID != m3.ID {
		t.Fatalf("delete template result = %+v / %+v", template, memes)
	}
	if deleted, err := st.IsTemplateDeleted("t2"); err != nil || !deleted {
		t.Fatalf("tombstone = %v err=%v", deleted, err)
	}
	if _, err := st.GetMeme(m3.ID, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("meme from deleted template still exists: %v", err)
	}

	if _, err := st.DeleteMeme(m2.ID); err != nil {
		t.Fatalf("delete second meme: %v", err)
	}
}

func TestMigrateSpecHelpers(t *testing.T) {
	st := testStore(t)
	mustTemplate(t, st, "t", "/uploads/templates/t.png")
	stored, err := st.CreateMeme(Meme{
		Title:      "stored",
		ImageURL:   "/uploads/memes/stored.png",
		TemplateID: "t",
		TopText:    "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	memes, err := st.ListStoredMemes()
	if err != nil || len(memes) != 1 || memes[0].ID != stored.ID {
		t.Fatalf("stored memes = %+v err=%v", memes, err)
	}
	if err := st.ClearMemeImage(stored.ID); err != nil {
		t.Fatal(err)
	}
	memes, err = st.ListStoredMemes()
	if err != nil || len(memes) != 0 {
		t.Fatalf("after clear = %+v err=%v", memes, err)
	}
	got, err := st.GetMeme(stored.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ImageURL != "/api/memes/"+stored.ID+"/image" || got.StoredImage != "" {
		t.Fatalf("virtual image = %q stored=%q", got.ImageURL, got.StoredImage)
	}
	if n, err := st.CountMemesByImageURL("/uploads/memes/stored.png"); err != nil || n != 0 {
		t.Fatalf("image references = %d err=%v", n, err)
	}
}
