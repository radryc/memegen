// Command memebank-admin manages a local memebank data directory.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/radryc/memebank/server/internal/store"
)

func main() {
	dataDir := flag.String("data", "data", "data directory containing memebank.db and uploads/")
	flag.Usage = usage
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	dbPath := filepath.Join(*dataDir, "memebank.db")
	if _, err := os.Stat(dbPath); err != nil {
		fatalf("database not found at %s\nUse -data to point at the server's data directory.", dbPath)
	}
	st, err := store.Open(dbPath)
	if err != nil {
		fatalf("open database: %v", err)
	}
	defer st.Close()

	switch args[0] {
	case "templates":
		cmdTemplates(st, *dataDir, args[1:])
	case "memes":
		cmdMemes(st, *dataDir, args[1:])
	case "stats":
		cmdStats(st)
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `memebank-admin — manage a local memebank database

Usage:
  memebank-admin [-data DIR] <command> [options]

Commands:
  templates list  [-q TEXT]
  templates delete ID [-with-memes] [-yes]
  memes list      [-q TEXT] [-template ID] [-limit N]
  memes delete ID [-yes]
  memes migrate-spec [-yes]
  stats

Notes:
  • Deletes are local and immediate. -yes is required to actually delete.
  • Deleting a template that has memes requires -with-memes.
  • Deleted templates are remembered so seeding will not recreate them.
  • migrate-spec converts legacy stored meme images into text specs and removes
    their files when the template and captions are still available.
`)
}

func cmdStats(st *store.Store) {
	stats, err := st.Stats()
	if err != nil {
		fatalf("stats: %v", err)
	}
	fmt.Printf("memes:     %d\nvotes:     %d\ntemplates: %d\n", stats.Memes, stats.Votes, stats.Templates)
}

func cmdTemplates(st *store.Store, dataDir string, args []string) {
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("templates list", flag.ExitOnError)
		q := fs.String("q", "", "filter by name")
		_ = fs.Parse(args[1:])
		templates, err := st.ListTemplates("", *q, "")
		if err != nil {
			fatalf("list templates: %v", err)
		}
		counts, err := st.MemeCountsByTemplate()
		if err != nil {
			fatalf("count memes: %v", err)
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tNAME\tSOURCE\tTYPE\tMEMES")
		for _, template := range templates {
			kind := "image"
			if template.Animated {
				kind = "gif"
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\n", template.ID, truncate(template.Name, 45), template.Source, kind, counts[template.ID])
		}
		_ = tw.Flush()
		fmt.Printf("\n%d templates\n", len(templates))
	case "delete":
		id, withMemes, yes, parseErr := parseDeleteArgs(args[1:], true)
		if parseErr != nil {
			fatalf("%v\nusage: memebank-admin templates delete ID [-with-memes] [-yes]", parseErr)
		}
		template, err := st.GetTemplate(id)
		if errors.Is(err, store.ErrNotFound) {
			fatalf("template %q not found", id)
		}
		if err != nil {
			fatalf("lookup template: %v", err)
		}
		count, err := st.CountMemesByTemplate(id)
		if err != nil {
			fatalf("count memes: %v", err)
		}
		if count > 0 && !withMemes {
			fatalf("template %q (%s) is used by %d meme(s).\nRe-run with -with-memes to delete those memes too.", id, template.Name, count)
		}
		if !yes {
			fmt.Printf("Would delete template %q (%s)", template.Name, template.ID)
			if count > 0 {
				fmt.Printf(" and %d meme(s) using it", count)
			}
			fmt.Println(".")
			fmt.Println("Nothing deleted. Re-run with -yes to confirm.")
			return
		}
		memes, deletedTemplate, err := st.DeleteTemplate(id, withMemes)
		if err != nil {
			fatalf("delete template: %v", err)
		}
		fmt.Printf("Deleted template %q (%s)\n", deletedTemplate.Name, deletedTemplate.ID)
		if err := removeIfUnused(st, dataDir, deletedTemplate.ImageURL, true); err != nil {
			fmt.Fprintf(os.Stderr, "warning: template image cleanup: %v\n", err)
		}
		for _, meme := range memes {
			if err := removeIfUnused(st, dataDir, meme.StoredImage, false); err != nil {
				fmt.Fprintf(os.Stderr, "warning: meme image cleanup (%s): %v\n", meme.ID, err)
			}
		}
		if len(memes) > 0 {
			fmt.Printf("Deleted %d meme(s) that used it.\n", len(memes))
		}
	default:
		usage()
		os.Exit(2)
	}
}

func cmdMemes(st *store.Store, dataDir string, args []string) {
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}
	switch args[0] {
	case "list":
		fs := flag.NewFlagSet("memes list", flag.ExitOnError)
		q := fs.String("q", "", "search title, captions, author or template")
		template := fs.String("template", "", "filter by template id")
		limit := fs.Int("limit", 50, "maximum rows (1-100)")
		_ = fs.Parse(args[1:])
		if *limit < 1 {
			*limit = 1
		}
		if *limit > 100 {
			*limit = 100
		}
		memes, _, err := st.ListMemes(store.ListOptions{
			Sort:     store.SortNew,
			Limit:    *limit,
			Search:   *q,
			Template: *template,
		})
		if err != nil {
			fatalf("list memes: %v", err)
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "ID\tSCORE\tAUTHOR\tSTORAGE\tTITLE")
		for _, meme := range memes {
			storage := "spec"
			if meme.StoredImage != "" {
				storage = "file"
				if meme.Animated {
					storage = "file/gif"
				}
			}
			fmt.Fprintf(tw, "%s\t%d\t%s\t%s\t%s\n", meme.ID, meme.Score, truncate(meme.Author, 18), storage, truncate(meme.Title, 50))
		}
		_ = tw.Flush()
		fmt.Printf("\nShowing %d (latest first)\n", len(memes))
	case "delete":
		id, _, yes, parseErr := parseDeleteArgs(args[1:], false)
		if parseErr != nil {
			fatalf("%v\nusage: memebank-admin memes delete ID [-yes]", parseErr)
		}
		meme, err := st.GetMeme(id, "")
		if errors.Is(err, store.ErrNotFound) {
			fatalf("meme %q not found", id)
		}
		if err != nil {
			fatalf("lookup meme: %v", err)
		}
		if !yes {
			fmt.Printf("Would delete meme %q by %s (%s).\n", meme.Title, meme.Author, meme.ID)
			fmt.Println("Nothing deleted. Re-run with -yes to confirm.")
			return
		}
		deleted, err := st.DeleteMeme(id)
		if err != nil {
			fatalf("delete meme: %v", err)
		}
		if err := removeIfUnused(st, dataDir, deleted.StoredImage, false); err != nil {
			fmt.Fprintf(os.Stderr, "warning: image cleanup: %v\n", err)
		}
		fmt.Printf("Deleted meme %q (%s).\n", deleted.Title, deleted.ID)
	case "migrate-spec":
		fs := flag.NewFlagSet("memes migrate-spec", flag.ExitOnError)
		yes := fs.Bool("yes", false, "actually migrate")
		_ = fs.Parse(args[1:])
		migrateSpec(st, dataDir, *yes)
	default:
		usage()
		os.Exit(2)
	}
}

func migrateSpec(st *store.Store, dataDir string, yes bool) {
	memes, err := st.ListStoredMemes()
	if err != nil {
		fatalf("list stored memes: %v", err)
	}
	type candidate struct {
		meme  store.Meme
		bytes int64
	}
	var migrate []candidate
	skipped := 0
	for _, meme := range memes {
		if meme.TemplateID == "" || meme.Width == 0 || meme.Height == 0 ||
			(meme.TopText == "" && meme.BottomText == "") {
			skipped++
			continue
		}
		var size int64
		if path := uploadPath(dataDir, meme.StoredImage); path != "" {
			if info, err := os.Stat(path); err == nil {
				size = info.Size()
			}
		}
		migrate = append(migrate, candidate{meme: meme, bytes: size})
	}
	if len(migrate) == 0 {
		fmt.Printf("Nothing to migrate (%d stored memes, %d skipped).\n", len(memes), skipped)
		return
	}
	var total int64
	for _, item := range migrate {
		total += item.bytes
	}
	fmt.Printf("Stored memes: %d, migratable: %d, skipped: %d\n", len(memes), len(migrate), skipped)
	fmt.Printf("Potential disk savings: %.1f KB\n", float64(total)/1024)
	if !yes {
		fmt.Println("Nothing changed. Re-run with -yes to migrate.")
		return
	}
	migrated := 0
	var freed int64
	for _, item := range migrate {
		if err := st.ClearMemeImage(item.meme.ID); err != nil {
			fmt.Fprintf(os.Stderr, "warning: clear image for %s: %v\n", item.meme.ID, err)
			continue
		}
		migrated++
		count, err := st.CountMemesByImageURL(item.meme.StoredImage)
		if err == nil && count == 0 {
			if path := uploadPath(dataDir, item.meme.StoredImage); path != "" {
				if info, statErr := os.Stat(path); statErr == nil {
					if removeErr := os.Remove(path); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
						fmt.Fprintf(os.Stderr, "warning: remove %s: %v\n", path, removeErr)
					} else {
						freed += info.Size()
					}
				}
			}
		}
	}
	fmt.Printf("Migrated %d meme(s); freed %.1f KB of rendered image files.\n", migrated, float64(freed)/1024)
}

func removeIfUnused(st *store.Store, dataDir, url string, template bool) error {
	if url == "" {
		return nil
	}
	var count int
	var err error
	if template {
		count, err = st.CountTemplatesByImageURL(url)
	} else {
		count, err = st.CountMemesByImageURL(url)
	}
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	path := uploadPath(dataDir, url)
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	fmt.Printf("  removed %s\n", filepath.ToSlash(path))
	return nil
}

func uploadPath(dataDir, url string) string {
	if url == "" || !strings.HasPrefix(url, "/uploads/") {
		return ""
	}
	root, err := filepath.Abs(filepath.Join(dataDir, "uploads"))
	if err != nil {
		return ""
	}
	path, err := filepath.Abs(filepath.Join(dataDir, filepath.FromSlash(strings.TrimPrefix(url, "/"))))
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
		return ""
	}
	return path
}

func parseDeleteArgs(args []string, allowWithMemes bool) (id string, withMemes, yes bool, err error) {
	for _, arg := range args {
		switch {
		case arg == "-yes" || arg == "--yes":
			yes = true
		case allowWithMemes && (arg == "-with-memes" || arg == "--with-memes"):
			withMemes = true
		case strings.HasPrefix(arg, "-"):
			return "", false, false, fmt.Errorf("unknown flag %s", arg)
		case id == "":
			id = arg
		default:
			return "", false, false, fmt.Errorf("unexpected argument %s", arg)
		}
	}
	if id == "" {
		return "", false, false, fmt.Errorf("an ID is required")
	}
	return id, withMemes, yes, nil
}

func truncate(value string, max int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:max-1]) + "…"
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
